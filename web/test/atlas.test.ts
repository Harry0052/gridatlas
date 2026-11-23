import assert from "node:assert/strict";
import test from "node:test";
import { announcement, valueText, viewRows, type Atlas, type MetricKey } from "../src/atlas.ts";

const atlas: Atlas = {
  schema_version: "gridatlas.atlas.v1",
  as_of: "2026-06-01",
  limitations: ["Headroom is unobserved."],
  regions: [
    {
      region_id: "SYN-NORTH",
      name: "North Planning Region (synthetic)",
      metrics: {
        installed_nameplate_mw: metric("available", "available", 120, false, "2026-03-15"),
        queued_requested_mw: metric("available", "available", 50, false, "2026-04-01"),
        headroom_mw: {
          status: "unobserved",
          display_status: "unobserved",
          value_mw: null,
          zero: false,
          unit: "MW",
          definition: "Unobserved.",
          reason: "not_estimated",
        },
      },
    },
    {
      region_id: "SYN-SOUTH",
      name: "South Planning Region (synthetic)",
      metrics: {
        installed_nameplate_mw: metric("missing", "missing", null, false, ""),
        queued_requested_mw: metric("incompatible", "incompatible", null, false, "2026-05-01", "mislabeled_unit"),
        headroom_mw: {
          status: "unobserved",
          display_status: "unobserved",
          value_mw: null,
          zero: false,
          unit: "MW",
          definition: "Unobserved.",
          reason: "not_estimated",
        },
      },
    },
    {
      region_id: "SYN-CENTRAL",
      name: "Central Planning Region (synthetic)",
      metrics: {
        installed_nameplate_mw: metric("available", "available", 0, true, "2026-05-01"),
        queued_requested_mw: metric("stale", "stale", 40, false, "2025-01-01"),
        headroom_mw: {
          status: "unobserved",
          display_status: "unobserved",
          value_mw: null,
          zero: false,
          unit: "MW",
          definition: "Unobserved.",
          reason: "not_estimated",
        },
      },
    },
  ],
};

function metric(
  status: string,
  display: string,
  value: number | null,
  zero: boolean,
  release: string,
  reason = "",
) {
  return {
    status,
    display_status: display,
    value_mw: value,
    zero,
    unit: "MW",
    definition: "definition",
    release_date: release,
    reason,
    coverage: {
      included_records: 1,
      duplicate_rows_ignored: 0,
      omitted_rows: 0,
      withdrawn_rows: 0,
      ambiguous_rows: 0,
      ambiguous_requests: 0,
      included_record_ids: ["A"],
    },
    provenance: { as_of: "2026-06-01", snapshot_file: "snapshots/example.csv" },
    trace: [],
  };
}

test("filters keep installed, queued, and headroom distinct", () => {
  const before = JSON.stringify(atlas);
  const installed = viewRows(atlas, "installed_nameplate_mw");
  const queued = viewRows(atlas, "queued_requested_mw");
  const headroom = viewRows(atlas, "headroom_mw");
  assert.equal(JSON.stringify(atlas), before);
  assert.equal(installed.find((row) => row.regionId === "SYN-NORTH")?.valueMW, 120);
  assert.equal(queued.find((row) => row.regionId === "SYN-NORTH")?.valueMW, 50);
  assert.equal(headroom.find((row) => row.regionId === "SYN-NORTH")?.valueMW, null);
  assert.equal(valueText(headroom[0]), "unobserved");
  assert.equal(installed.find((row) => row.regionId === "SYN-SOUTH")?.valueMW, null);
  assert.equal(valueText(installed.find((row) => row.regionId === "SYN-CENTRAL")!), "0 MW (reported zero)");
  assert.equal(queued.find((row) => row.regionId === "SYN-CENTRAL")?.displayStatus, "stale");
  assert.match(announcement("queued_requested_mw", queued), /Queued requested MW/);
  const keys: MetricKey[] = ["installed_nameplate_mw", "queued_requested_mw", "headroom_mw"];
  assert.equal(keys.length, 3);
});
