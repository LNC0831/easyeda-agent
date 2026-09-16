package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

// Deliberately asymmetric, synthetic geometry: the closest legal R2 placement
// prevents R3's naming corridor. Releasing R2's checkpoint relocates it and
// recomputes the R2/R3 routes; array permutation alone cannot pass this test.
func schematicRepairFixture() SchematicLayoutInput {
	return SchematicLayoutInput{SchemaVersion: 1, CoreComponentID: "core", MaxCandidates: 20000,
		NetPolicies: map[string]string{"G": "local_ground", "N0": "module_port", "N1": "module_port", "N2": "module_port"},
		Components: []SchematicLayoutComponent{
			{ID: "core", Measurement: SchematicPlacement{Designator: "U1", BBox: SchematicBox{-20, -40, 20, 40}, Pins: []SchematicPin{
				{Number: "4", Net: "G", X: 0, Y: -50}, {Number: "1", Net: "N0", X: 30, Y: 20}, {Number: "2", Net: "N1", X: 30, Y: 0}, {Number: "3", Net: "N2", X: 30, Y: -20}, {Number: "5", X: -30, Y: 20},
			}}, PinStates: map[string]string{"5": "nc"}},
			{ID: "R1", Measurement: SchematicPlacement{Designator: "R1", BBox: SchematicBox{-5, -20, 5, 20}, Pins: []SchematicPin{{Number: "1", Net: "N0", X: 0, Y: -30}, {Number: "2", Net: "G", X: 0, Y: 30}}}},
			{ID: "R2", Measurement: SchematicPlacement{Designator: "R2", Rotation: 180, Mirror: true, BBox: SchematicBox{-15, -15, 15, 15}, Pins: []SchematicPin{{Number: "1", Net: "N1", X: -25, Y: 0}, {Number: "2", Net: "G", X: 25, Y: 0}}}},
			{ID: "R3", Measurement: SchematicPlacement{Designator: "R3", BBox: SchematicBox{-10, -15, 10, 15}, Pins: []SchematicPin{{Number: "1", Net: "N2", X: 0, Y: 25}, {Number: "2", Net: "G", X: 0, Y: -25}}}},
		}}
}

func TestSchematicRepairMovesPreviouslyPlacedPeripheral(t *testing.T) {
	in := schematicRepairFixture()
	before, _ := json.Marshal(in)
	measured, members := map[string]powerLayoutPlacement{}, []string{}
	for _, c := range in.Components {
		measured[c.ID], members = c.Measurement, append(members, c.ID)
	}
	greedyBudget := in.MaxCandidates
	greedy := newSchematicRepairSearch(in, measured, members, nil, &greedyBudget)
	greedy.diagnostics.BranchLimit = 1 // First failing descendant; no relocation.
	if out, err := greedy.solve(powerLayoutPlan{Placements: []powerLayoutPlacement{measured["core"]}}, members[1:]); err == nil || out != nil {
		t.Fatal("fixture no longer exercises a failed greedy prefix", err)
	}
	if greedy.diagnostics.Backtracks > greedy.diagnostics.BranchLimit {
		t.Fatal("ancestor rollback incremented an exhausted branch limit")
	}
	out, err := PlanSchematicLayout(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Search == nil || out.Search.Backtracks == 0 || out.Search.RepairAttempts == 0 || !strings.Contains(strings.Join(out.Search.MovedComponents, ","), "R2") {
		t.Fatalf("missing actual relocation diagnostics: %+v", out.Search)
	}
	for i, c := range out.Placements {
		m := measured[out.ComponentIDs[c.Designator]]
		if c.Rotation != m.Rotation || c.Mirror != m.Mirror || len(c.Pins) != len(m.Pins) {
			t.Fatal("changed measured pose/pin membership")
		}
		if c.Designator == "R2" && greedy.firstXY["R2"] == [2]float64{c.X, c.Y} {
			t.Fatal("previously placed R2 was not moved")
		}
		for j, q := range c.Pins {
			if q.Number != m.Pins[j].Number || q.Net != m.Pins[j].Net || q.X-c.X != m.Pins[j].X-m.X || q.Y-c.Y != m.Pins[j].Y-m.Y {
				t.Fatal("changed pin/net/relative geometry")
			}
		}
		if i == 0 && (c.Designator != "U1" || c.X != 0 || c.Y != 0) {
			t.Fatal("moved core")
		}
	}
	if out.PinStates["core"]["5"] != "nc" {
		t.Fatal("lost NC")
	}
	p := powerLayoutPlan{Placements: out.Placements, Wires: out.Wires, Flags: out.Flags}
	if err = validateLibGeometry(&p); err != nil {
		t.Fatal(err)
	}
	if err = validateSchCompositionNets(&p); err != nil {
		t.Fatal("repaired suffix has invalid or stale routes", err)
	}
	again, err := PlanSchematicLayout(in)
	if err != nil || !reflect.DeepEqual(out, again) {
		t.Fatal("search is not deterministic", err)
	}
	after, _ := json.Marshal(in)
	if !bytes.Equal(before, after) {
		t.Fatal("mutated source evidence")
	}
	if out.CandidatesUsed > in.MaxCandidates || out.CandidatesUsed <= 0 {
		t.Fatal("invalid shared budget accounting")
	}
}

func TestRepairBranchLimitUsesSharedCandidateAllowance(t *testing.T) {
	in := schematicRepairFixture()
	measured := map[string]powerLayoutPlacement{}
	members := make([]string, 0, len(in.Components))
	for _, component := range in.Components {
		measured[component.ID] = component.Measurement
		members = append(members, component.ID)
	}
	budget := 4096
	search := newSchematicRepairSearch(in, measured, members, nil, &budget)
	if search.diagnostics.BranchLimit != budget {
		t.Fatalf("branch cap discarded shared search capacity: got %d want %d", search.diagnostics.BranchLimit, budget)
	}
	budget = 7
	search = newSchematicRepairSearch(in, measured, members, nil, &budget)
	if search.diagnostics.BranchLimit != 128 {
		t.Fatalf("tiny-budget secondary guard changed: got %d want 128", search.diagnostics.BranchLimit)
	}
}

func TestRouteConflictIncludesWireOwnersAndKeepsUnknownOwnersSearchable(t *testing.T) {
	a := powerLayoutPlacement{Designator: "A", BBox: layoutBBox{-120, -20, -80, 20}, Pins: []powerLayoutPin{{Number: "1", Net: "N", X: -70, Y: 0, Rotation: directionNumber(0)}}}
	b := powerLayoutPlacement{Designator: "B", BBox: layoutBBox{80, -20, 120, 20}, Pins: []powerLayoutPin{{Number: "1", Net: "N", X: 70, Y: 0}}}
	owner := powerLayoutPlacement{Designator: "Q", BBox: layoutBBox{190, 190, 210, 210}, Pins: []powerLayoutPin{{Number: "1", Net: "BLOCK", X: 180, Y: 200}}}
	// Enclose A in a finite, physically contacted wire cage. Unlike the former
	// open wall, this cannot be escaped by routing around an endpoint. Teeth on
	// every 5-raw crossing turn a would-be X into an endpoint/T/overlap contact;
	// the proper-X rule itself remains unchanged.
	barriers := []powerLayoutWire{
		{Net: "BLOCK", Points: [][2]float64{{-140, -50}, {-40, -50}}},
		{Net: "BLOCK", Points: [][2]float64{{-40, -50}, {-40, 50}}},
		{Net: "BLOCK", Points: [][2]float64{{-40, 50}, {-140, 50}}},
		{Net: "BLOCK", Points: [][2]float64{{-140, 50}, {-140, -50}}},
	}
	for y := -50.0; y <= 50; y += 5 {
		barriers = append(barriers,
			powerLayoutWire{Net: "BLOCK", Points: [][2]float64{{-40, y}, {-35, y}}},
			powerLayoutWire{Net: "BLOCK", Points: [][2]float64{{-140, y}, {-145, y}}},
		)
	}
	for x := -140.0; x <= -40; x += 5 {
		barriers = append(barriers,
			powerLayoutWire{Net: "BLOCK", Points: [][2]float64{{x, 50}, {x, 55}}},
			powerLayoutWire{Net: "BLOCK", Points: [][2]float64{{x, -50}, {x, -55}}},
		)
	}
	for _, known := range []bool{true, false} {
		p := powerLayoutPlan{Placements: []powerLayoutPlacement{a, b}, Wires: barriers}
		if known {
			p.Placements = append(p.Placements, owner)
		}
		err := libJoinNetsMode(&p, map[string]string{"N": "direct", "BLOCK": "module_port"}, false, false)
		var conflict *schematicRouteConflict
		if !errors.As(err, &conflict) || conflict.net != "N" || conflict.ownersComplete != known {
			t.Fatalf("known=%v: missing structured route conflict/ownership: %v", known, err)
		}
		if known && !conflict.blockers["Q"] {
			t.Fatal("ignored a component whose wire, but not body, blocks the route")
		}
		if !conflict.endpointOwners["A"] || !conflict.endpointOwners["B"] {
			t.Fatalf("lost physical-island endpoint owners: %+v", conflict.endpointOwners)
		}
		s := schematicRepairSearch{measured: map[string]powerLayoutPlacement{"owner": owner, "other": {Designator: "R", Pins: []powerLayoutPin{{Net: "OTHER"}}}}}
		if !s.participates("owner", conflict) || (!known && !s.participates("other", conflict)) {
			t.Fatal("unsafe pruning for real or unknown wire owners")
		}
		s.focused = true
		if !known && !s.participates("other", conflict) {
			t.Fatal("focused pass pruned unknown wire ownership")
		}
	}
}

func TestFocusedRouteConflictUsesActualBlockerBeforeEndpointFallback(t *testing.T) {
	conflict := &schematicRouteConflict{net: "N", endpointOwners: map[string]bool{"A": true, "B": true}, blockers: map[string]bool{"Q": true}, ownersComplete: true}
	s := schematicRepairSearch{focused: true, measured: map[string]powerLayoutPlacement{
		"a": {Designator: "A", Pins: []powerLayoutPin{{Net: "N"}}},
		"c": {Designator: "C", Pins: []powerLayoutPin{{Net: "N"}}},
		"q": {Designator: "Q", Pins: []powerLayoutPin{{Net: "BLOCK"}}},
	}}
	if s.participates("a", conflict) || s.participates("c", conflict) || !s.participates("q", conflict) {
		t.Fatal("focused pass did not isolate actual rejected-edge blockers")
	}
	s.focused = false
	if !s.participates("a", conflict) || s.participates("c", conflict) || !s.participates("q", conflict) {
		t.Fatal("full pass did not combine endpoint and actual blocker owners")
	}
}

func TestTargetedRelocationExpandsPrimaryAxisWithinBound(t *testing.T) {
	p := powerLayoutPlan{Placements: []powerLayoutPlacement{
		{Designator: "U1", X: 0, Y: 0},
		{Designator: "J1", X: -20, Y: 5},
	}}
	deltas := schematicTargetedRelocationDeltas(&p, "J1", "U1")
	if len(deltas) != 32 || deltas[0] != [2]float64{-5, 0} || deltas[1] != [2]float64{-10, 0} || deltas[6] != [2]float64{-35, 0} || deltas[7] != [2]float64{-40, 0} {
		t.Fatal("primary outward relocation shell is incomplete or unordered", deltas)
	}
	for _, delta := range deltas {
		if math.Abs(delta[0]) > 40 || math.Abs(delta[1]) > 40 || (delta[0] != 0 && delta[1] != 0) {
			t.Fatal("relocation escaped its bounded cardinal shell", delta)
		}
	}
}

func TestSchematicRepairFailureKeepsSharedBudgetAndNoPartialResult(t *testing.T) {
	in := schematicRepairFixture()
	before, _ := json.Marshal(in)
	for _, maximum := range []int{1, 7, 128, 512} {
		budget := maximum
		out, err := planSchematicLayoutWithBudget(in, &budget)
		if err == nil || out != nil || !errors.Is(err, errLibLayoutBudget) {
			t.Fatalf("max=%d: invalid failure result %+v, %v", maximum, out, err)
		}
		if budget < 0 || budget >= maximum || !strings.Contains(err.Error(), "no capacity proof") {
			t.Fatal("budget reset/underflow or ambiguous failure", budget, err)
		}
	}
	after, _ := json.Marshal(in)
	if !bytes.Equal(before, after) {
		t.Fatal("failed search changed evidence")
	}
}

func TestAttachmentPairsShareDistanceShellBudget(t *testing.T) {
	core := powerLayoutPlacement{Designator: "U1", BBox: layoutBBox{-20, -20, 20, 20}, Pins: []powerLayoutPin{{Number: "L", Net: "N", X: -30, Y: 0}, {Number: "R", Net: "N", X: 30, Y: 0}}}
	// Leave the core's required 5raw outward exit clear; the wall still
	// blocks attachment on this side without making every trial invalid.
	wall := powerLayoutPlacement{Designator: "X1", BBox: layoutBBox{-60, -300, -36, 300}, TextBBoxes: []layoutBBox{{-60, -10, -50, 10}}, Pins: []powerLayoutPin{{Number: "1", X: -70, Y: 0}}}
	part := powerLayoutPlacement{Designator: "R1", BBox: layoutBBox{-5, -5, 5, 5}, Pins: []powerLayoutPin{{Number: "1", Net: "N", X: -15, Y: 0}, {Number: "2", Net: "G", X: 15, Y: 0}}}
	current := powerLayoutPlan{Placements: []powerLayoutPlacement{core, wall}}
	bad := libAttachmentPair{host: core.Pins[0], own: part.Pins[0], side: "left"}
	good := libAttachmentPair{host: core.Pins[1], own: part.Pins[0], side: "right"}
	policies := map[string]string{"N": "local_power", "G": "local_ground"}
	budget := 1000
	if out, err := libPlacePeripheral(current, part, bad, policies, &budget); out != nil || err == nil {
		t.Fatal("first pair should be blocked by the fixed wall")
	}
	budget = 1000
	out, err := libPlacePeripheralPairs(current, part, []libAttachmentPair{bad, good}, policies, &budget, nil)
	if err != nil || out == nil || out.Placements[2].X <= 0 || budget <= 0 {
		t.Fatalf("first blocked pair monopolized the budget: %v; remaining=%d", err, budget)
	}
}

func TestPlacementAlternativeCursorAdvancesDistanceShell(t *testing.T) {
	core := powerLayoutPlacement{Designator: "U1", BBox: layoutBBox{-20, -20, 20, 20}, Pins: []powerLayoutPin{{Number: "1", Net: "N", X: 30, Y: 0, Rotation: directionNumber(0)}}}
	part := powerLayoutPlacement{Designator: "R1", BBox: layoutBBox{-10, -10, 10, 10}, Pins: []powerLayoutPin{{Number: "1", Net: "N", X: -20, Y: 0, Rotation: directionNumber(180)}}}
	pair := libAttachmentPair{host: core.Pins[0], own: part.Pins[0], side: "right"}
	budget, cursor := 1000, 5.0
	first, err := libPlacePeripheralPairsWithRouting(powerLayoutPlan{Placements: []powerLayoutPlacement{core}}, part, []libAttachmentPair{pair}, map[string]string{"N": "module_port"}, &budget, nil, nil, &cursor)
	if err != nil || first == nil || cursor <= 5 {
		t.Fatalf("first shell did not advance cursor: cursor=%g err=%v", cursor, err)
	}
	secondStart := cursor
	firstXY := [2]float64{first.Placements[1].X, first.Placements[1].Y}
	second, err := libPlacePeripheralPairsWithRouting(powerLayoutPlan{Placements: []powerLayoutPlacement{core}}, part, []libAttachmentPair{pair}, map[string]string{"N": "module_port"}, &budget, map[[2]float64]bool{firstXY: true}, nil, &cursor)
	if err != nil || second == nil || cursor != secondStart+5 {
		t.Fatalf("second shell did not advance cursor: cursor=%g err=%v", cursor, err)
	}
	if second.Placements[1].Pins[0].X-first.Placements[1].Pins[0].X != 5 {
		t.Fatalf("checkpoint alternative stayed in the exhausted shell: first=%+v second=%+v", first.Placements[1], second.Placements[1])
	}
}

func TestAttachmentDepthKeepsOwnedPeripheralClusterTogether(t *testing.T) {
	hints := map[string]SchematicLayoutPeripheral{
		"host-a":  {ComponentID: "host-a", AttachTo: &SchematicLayoutAttach{ComponentID: "core"}},
		"child-a": {ComponentID: "child-a", AttachTo: &SchematicLayoutAttach{ComponentID: "host-a"}},
		"leaf-a":  {ComponentID: "leaf-a", AttachTo: &SchematicLayoutAttach{ComponentID: "child-a"}},
		"host-b":  {ComponentID: "host-b", AttachTo: &SchematicLayoutAttach{ComponentID: "core"}},
	}
	depth := schematicAttachmentDepths("core", []string{"core", "host-a", "child-a", "leaf-a", "host-b"}, hints)
	if depth["core"] != 0 || depth["host-a"] != 1 || depth["child-a"] != 2 || depth["leaf-a"] != 3 || depth["host-b"] != 1 {
		t.Fatalf("unexpected attachment depths: %+v", depth)
	}
}

func TestConnectedPinCountIsNameAndOrderIndependent(t *testing.T) {
	dense := powerLayoutPlacement{Designator: "J9", Pins: []powerLayoutPin{{Net: "A"}, {Net: "B"}, {Net: ""}, {Net: "C"}}}
	small := powerLayoutPlacement{Designator: "D1", Pins: []powerLayoutPin{{Net: "A"}, {Net: "B"}}}
	if libConnectedPinCount(dense) != 3 || libConnectedPinCount(small) != 2 {
		t.Fatal("constraint rank depends on something other than connected source pins")
	}
	dense.Designator, small.Designator = "X1", "Z99"
	if libConnectedPinCount(dense) <= libConnectedPinCount(small) {
		t.Fatal("renaming changed constrained-first ordering")
	}
}

func TestExplicitDenseAttachmentHostRanksBeforeAutomaticOrSparseHost(t *testing.T) {
	measured := map[string]powerLayoutPlacement{
		"dense":  {Designator: "X1", Pins: []powerLayoutPin{{Net: "A"}, {Net: "B"}, {Net: "C"}}},
		"sparse": {Designator: "X2", Pins: []powerLayoutPin{{Net: "A"}}},
	}
	hints := map[string]SchematicLayoutPeripheral{
		"dense-child":  {ComponentID: "dense-child", AttachTo: &SchematicLayoutAttach{ComponentID: "dense"}},
		"sparse-child": {ComponentID: "sparse-child", AttachTo: &SchematicLayoutAttach{ComponentID: "sparse"}},
		"automatic":    {ComponentID: "automatic"},
	}
	if got := schematicAttachmentHostPinCount("dense-child", measured, hints); got != 3 {
		t.Fatalf("dense host rank=%d", got)
	}
	if schematicAttachmentHostPinCount("dense-child", measured, hints) <= schematicAttachmentHostPinCount("sparse-child", measured, hints) || schematicAttachmentHostPinCount("automatic", measured, hints) != 0 {
		t.Fatal("host constraint rank depends on source order instead of explicit measured ownership")
	}
}
