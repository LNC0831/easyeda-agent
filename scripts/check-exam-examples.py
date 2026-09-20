#!/usr/bin/env python3
"""Check the 260919 source examples and recorded live evidence.

The checker keeps source transcription, offline checks, representative live
verification, and unfinished work separate.  It validates the summaries that
make an example teachable; it does not infer completion of untested PCB work.
"""

from __future__ import annotations

import json
import math
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
EXAMPLE = ROOT / "skills/easyeda-agent/references/examples/260919-at32f415"


def load(name: str) -> dict:
    with (EXAMPLE / name).open(encoding="utf-8") as stream:
        return json.load(stream)


def pin_map(connectivity: dict, reference: str) -> dict[str, dict]:
    component = next(c for c in connectivity["components"] if c["reference"] == reference)
    return {pin["terminal"]: pin for pin in component["pins"]}


def require_net(connectivity: dict, reference: str, terminal: str, net: str) -> None:
    actual = pin_map(connectivity, reference)[terminal].get("net")
    assert actual == net, f"{reference}.{terminal}: {actual!r} != {net!r}"


def require_nc(connectivity: dict, reference: str, terminal: str) -> None:
    actual = pin_map(connectivity, reference)[terminal].get("state")
    assert actual == "nc", f"{reference}.{terminal}: expected NC, got {actual!r}"


def close(actual: float, expected: float, *, tolerance: float = 0.01) -> bool:
    return math.isclose(actual, expected, rel_tol=0, abs_tol=tolerance)


def strings(value: object) -> list[str]:
    """Flatten JSON strings for execution-policy checks."""
    if isinstance(value, str):
        return [value]
    if isinstance(value, list):
        return [text for item in value for text in strings(item)]
    if isinstance(value, dict):
        return [text for item in value.values() for text in strings(item)]
    return []


def main() -> None:
    manifest = load("source-manifest.json")
    bom = load("bom-instances.json")
    connectivity = load("source-connectivity.json")
    catalog = load("example-catalog.json")
    placement = load("initial-placement.json")
    live = load("live-validation.json")

    refs = [item["reference"] for item in bom["instances"]]
    assert bom["counts"] == {"bomRows": 27, "instances": 69, "uniqueReferences": 69}
    assert len(refs) == len(set(refs)) == 69
    assert sum(row["quantity"] for row in bom["rows"]) == 69
    for row in bom["rows"]:
        assert row["quantity"] == len(row["references"]), row

    zone_refs = [ref for zone in manifest["functionalZones"] for ref in zone["refs"]]
    assert len(zone_refs) == len(set(zone_refs)) == 69
    assert set(zone_refs) == set(refs)
    assert all(item["functionalZone"] for item in bom["instances"])

    zone_by_ref = {
        ref: zone["id"]
        for zone in manifest["functionalZones"]
        for ref in zone["refs"]
    }
    bom_by_ref = {item["reference"]: item for item in bom["instances"]}
    for ref, item in bom_by_ref.items():
        assert item["functionalZone"] == zone_by_ref[ref], ref

    conn_refs = [item["reference"] for item in connectivity["components"]]
    assert connectivity["status"] == "source-only"
    assert len(conn_refs) == len(set(conn_refs)) == 69
    assert set(conn_refs) == set(refs)
    defined_nets = set(connectivity["netDefinitions"])
    used_nets: set[str] = set()
    pin_count = 0
    for component in connectivity["components"]:
        reference = component["reference"]
        source = bom_by_ref[reference]
        assert component["functionalZone"] == zone_by_ref[reference], reference
        assert component["value"] == source["value"], reference
        assert component["footprint"] == source["footprint"], reference
        assert isinstance(component["pinNumbersFromSource"], bool), reference
        terminals: set[str] = set()
        for pin in component["pins"]:
            pin_count += 1
            terminal = pin["terminal"]
            assert terminal not in terminals, f"{reference}: duplicate terminal {terminal}"
            terminals.add(terminal)
            has_net = "net" in pin
            has_state = "state" in pin
            assert has_net != has_state, f"{reference}.{terminal}: need exactly one of net/state"
            if has_net:
                used_nets.add(pin["net"])
            else:
                assert pin["state"] == "nc", f"{reference}.{terminal}: unsupported state"
    assert pin_count == 233
    assert len(defined_nets) == 46
    assert used_nets == defined_nets
    explicit_nc = sum(
        pin.get("state") == "nc"
        for component in connectivity["components"]
        for pin in component["pins"]
    )
    assert connectivity["counts"]["explicitNC"] == explicit_nc == 13

    # High-risk transcription points named by the exam source review.
    for terminal in ("2", "4"):
        require_net(connectivity, "U2", terminal, "+3V3")
    for terminal in ("5", "8"):
        require_net(connectivity, "U4", terminal, "+3V3")
    require_nc(connectivity, "U4", "4")
    for terminal in ("B6", "A6"):
        require_net(connectivity, "USB1", terminal, "USB_D+")
    for terminal in ("A7", "B7"):
        require_net(connectivity, "USB1", terminal, "USB_D-")
    for terminal in ("A8", "B8"):
        require_nc(connectivity, "USB1", terminal)
    require_nc(connectivity, "BUZZER1", "NC")
    for terminal in ("1", "2", "9"):
        require_nc(connectivity, "U3", terminal)
    require_nc(connectivity, "U5", "5")
    for terminal in ("10", "11", "12", "13"):
        require_net(connectivity, "CARD1", terminal, "GND")
    for ref in ("SCREW1", "SCREW2", "SCREW3", "SCREW4"):
        require_nc(connectivity, ref, "1")

    fixed = {item["ref"]: item for item in manifest["fixedPlacementsMm"]}
    expected = {
        "SCREW1": (3, 47, 0), "SCREW2": (87, 47, 0),
        "SCREW3": (3, 3, 0), "SCREW4": (87, 3, 0),
        "U6": (45, 25, 0), "CARD1": (79.5, 25, 90),
    }
    for ref, (x, y, rotation) in expected.items():
        assert (fixed[ref]["x"], fixed[ref]["y"], fixed[ref]["rotationDeg"]) == (x, y, rotation)
        assert fixed[ref]["locked"] is True
    assert fixed["CN1"]["x"] is None and fixed["CN1"]["y"] == 42
    assert fixed["CN1"]["rotationDeg"] == 180 and fixed["CN1"]["freeAxis"] == "x"

    expected_catalog_ids = {
        *(f"SCH-{i:02d}" for i in range(1, 11)),
        *(f"PCB-{i:02d}" for i in range(1, 9)),
        *(f"LAY-{i:02d}" for i in range(1, 7)),
        *(f"RTE-{i:02d}" for i in range(1, 9)),
        *(f"FIN-{i:02d}" for i in range(1, 5)),
    }
    catalog_entries = {entry["id"]: entry for entry in catalog["entries"]}
    assert set(catalog_entries) == expected_catalog_ids
    assert catalog["status"] == "partial-live-verified"
    assert catalog["counts"] == {
        "entries": 36, "SCH": 10, "PCB": 8, "LAY": 6, "RTE": 8, "FIN": 4,
    }
    assert catalog["executionPolicy"] == {
        "allowed": ["easyeda Cobra subcommand", "typed action", "easyeda apply"],
        "forbidden": [
            "GUI/CUA", "mouse/keyboard/canvas", "property panel/project tree",
            "manual design repair", "debug.exec_js design mutation",
        ],
        "missingCapability": (
            "mark planned/unsupported; implement and validate a typed interface "
            "before live mutation"
        ),
    }
    forbidden_execution_terms = (
        "gui", "cua", "属性面板", "工程树", "刷新浏览器", "刷新整个内置浏览器",
        "mouse", "keyboard", "canvas", "property panel", "project tree",
    )
    executable_text = "\n".join(strings({
        "stepTemplates": catalog["stepTemplates"],
        "entries": catalog["entries"],
    })).lower()
    for term in forbidden_execution_terms:
        assert term.lower() not in executable_text, (
            f"example execution path must not contain interactive fallback: {term}"
        )
    assert {"initial-placement.json", "live-validation.json"} <= set(catalog["sourceData"])
    required_example_fields = {
        "source", "problem", "startState", "parameters", "steps", "commands",
        "observations", "rationale", "knownErrorsAndFixes", "verificationStatus", "pending",
    }
    allowed_statuses = {"source-only", "offline-verified", "live-verified"}
    for example_id, entry in catalog_entries.items():
        assert required_example_fields <= entry.keys(), example_id
        status = entry["verificationStatus"]
        assert status in allowed_statuses, example_id
        for field in ("source", "parameters", "steps", "commands", "observations", "knownErrorsAndFixes"):
            assert entry[field], f"{example_id}: empty {field}"

        if status == "live-verified":
            assert entry["pending"] == [], f"{example_id}: live-verified still has pending work"
            assert isinstance(entry.get("liveEvidence"), str) and entry["liveEvidence"].strip(), (
                f"{example_id}: live-verified requires concrete liveEvidence"
            )
        else:
            assert entry["pending"], f"{example_id}: unfinished status needs explicit pending work"

    assert placement["status"] == "partial-live-verified"
    for field in (
        "source", "startState", "parameters", "commands", "observations",
        "knownErrorsAndFixes", "evidence", "independentVerification", "notClaimed",
    ):
        assert placement[field], f"initial-placement.json: empty {field}"
    assert placement["independentVerification"]["status"] == "completed-with-findings"
    assert placement["units"] == "mil"
    assert placement["coordinateSemantic"] == "footprint-anchor"
    assert placement["board"] == {
        "widthMil": 3543.31, "heightMil": 1968.5, "widthMm": 90, "heightMm": 50,
    }
    placements = {item["ref"]: item for item in placement["placements"]}
    assert len(placements) == len(placement["placements"]) == 69
    assert set(placements) == set(refs)
    for ref, (x_mm, y_mm, rotation) in expected.items():
        item = placements[ref]
        assert close(item["xMil"], x_mm / 0.0254), ref
        assert close(item["yMil"], y_mm / 0.0254), ref
        assert item["rotationDeg"] == rotation, ref
        assert item["locked"] is True, ref
    cn1 = placements["CN1"]
    assert close(cn1["yMil"], 42 / 0.0254)
    assert cn1["rotationDeg"] == 180 and cn1["locked"] is False
    placement_verification = placement["verification"]
    assert placement_verification["componentCount"] == 69
    assert placement_verification["uniqueDesignators"] == 69
    for field in (
        "placementDiffCountAfterFullBrowserReload", "bboxCollisionCount",
        "sub6MilBBoxPairCount", "layoutLintOutsideOutlineCount",
        "layoutLintTightSpacingCount",
    ):
        assert placement_verification[field] == 0, field

    assert live["status"] == "partial-live-verified"
    assert live["purpose"].endswith("不是完成态整板答案")
    schematic = live["schematic"]
    assert (schematic["components"], schematic["terminals"], schematic["explicitNc"], schematic["nets"]) == (
        69, 233, 13, 46,
    )
    assert schematic["sourceEndpointDiffs"] == 0
    assert schematic["officialDrc"]["total"] == 0
    mechanics = live["pcbMechanics"]
    assert mechanics["components"] == 69
    outline = mechanics["outline"]
    assert close(outline["widthMil"], 90 / 0.0254)
    assert close(outline["heightMil"], 50 / 0.0254)
    assert close(outline["radiusMil"], 3 / 0.0254)
    assert outline["nativeArcs"] == 4 and outline["locked"] is True
    rules = live["pcbRules"]
    assert rules["signalTrackMil"] == {"min": 8.0, "default": 8.0}
    assert rules["clearanceMil"] == 6.0
    assert rules["viaMil"] == {"outerMin": 24.0, "holeMin": 12.0}
    assert rules["powerTrackMil"] == {"min": 8.0, "default": 20.0}
    assert set(rules["netClass"]["nets"]) == {"+5V", "+3V3", "GND"}
    assert rules["verifiedAfterFullBrowserReload"] is True
    assert live["notClaimed"], "partial live validation must list unverified work"

    live_count = sum(entry["verificationStatus"] == "live-verified" for entry in catalog_entries.values())
    print(
        "260919 examples: 27 BOM rows, 69 instances, 69 connectivity records, "
        f"233 terminals, 46 nets, 15 zones, 36 catalog entries, {live_count} live-verified — consistent"
    )


if __name__ == "__main__":
    main()
