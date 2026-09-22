package app

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func newProjectImportCmd(cfg *appConfig, window *string, stdout io.Writer) *cobra.Command {
	var source, input, team, name string
	var ack bool
	c := &cobra.Command{Use: "import", Short: "Import native epro2 into a NEW project; does not verify restoration", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(*window) == "" || strings.TrimSpace(source) == "" || strings.TrimSpace(team) == "" || strings.TrimSpace(name) == "" || !ack {
			return fmt.Errorf("--window, --project-uuid (active source), --team, --name and --allow-discard-unsaved are required")
		}
		if cfg.project != "" || cfg.doc != "" {
			return fmt.Errorf("omit global --project/--doc routing")
		}
		if !strings.EqualFold(filepath.Ext(input), ".epro2") {
			return fmt.Errorf("--file must be native .epro2")
		}
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, projectArchiveLimit+1))
		if err != nil {
			return err
		}
		if err = validateNativeProjectArchive(data); err != nil {
			return err
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		payload := map[string]any{"projectUuid": source, "teamUuid": team, "friendlyName": name, "allowDiscardUnsaved": true, "base64": base64.StdEncoding.EncodeToString(data), "sha256": digest}
		res, err := requestActionTimed(cfg, "project.import", *window, payload, 60*time.Second)
		if err != nil {
			return fmt.Errorf("native import failed (possible partial creation; inspect before retry): %w", err)
		}
		v := res.Result
		uuid, ok := v["uuid"].(string)
		if !ok || uuid == "" || uuid == source || v["identityVerified"] != true || v["sha256"] != digest || v["teamUuid"] != team || v["friendlyName"] != name {
			return fmt.Errorf("import response identity/owner/name/hash not verified; inspect before retry")
		}
		return encodeResultEnvelope(res, v, stdout)
	}}
	c.Flags().StringVar(&source, "project-uuid", "", "expected active SOURCE project UUID")
	c.Flags().StringVar(&input, "file", "", "native .epro2 file; bytes transported internally, never shell arguments")
	c.Flags().StringVar(&team, "team", "", "explicit owner team UUID")
	c.Flags().StringVar(&name, "name", "", "unique NEW project display name")
	c.Flags().BoolVar(&ack, "allow-discard-unsaved", false, "confirm all documents are saved before official import")
	return c
}
