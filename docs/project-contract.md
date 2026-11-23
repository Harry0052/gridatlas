# Project contract

Decision carried forward from `docs/overlap.md`: proceed with a separate local atlas. Do not wrap a hosted grid API, and do not ship facility locations, line routes, flow results, or an estimated headroom.

## User and task

A regional energy researcher compares public planning releases. They load a coarse regional bundle, set a frozen as-of date, and read one cell per region per metric. Each cell says whether the figure is available, stale, missing, or incompatible, shows the value only when the basis is defensible, and keeps the release date beside it. Headroom is always unobserved.

The shipped demo uses three synthetic planning regions (`SYN-NORTH`, `SYN-CENTRAL`, `SYN-SOUTH`) and two products: operating generator nameplate MW, and distinct active interconnection requests' requested MW.

## Minimum dependencies

- Go 1.22 or newer, standard library only, for ingestion, coverage scoring, JSON, and the text report. `CGO_ENABLED` may be on when the race detector runs; the program itself does not use cgo.
- Node.js 22 and TypeScript 5.6.3, development-only, to typecheck and compile the map. The page has no runtime package dependencies.
- No network access while scoring a bundle. No API key, account, or paid service.

Provenance path for the demo is the in-repository synthetic bundle: authored geometries, authored rows, SHA-256 of each snapshot, and a generation note in `testdata/fixtures/sample/MANIFEST.txt`. Real EIA-860M files are US government public domain and may be adapted by a user into the same CSV contract later. Berkeley Lab Queued Up files are not bundled; their redistribution terms stayed unresolved in P01.

## Acceptance case

Fixture: `testdata/fixtures/sample`. Tolerance: exact integer MW. This is a synthetic fixture, not a measurement of the physical grid.

Frozen as-of `2026-06-01`:

- `SYN-NORTH` installed nameplate is 120 MW, status available, release `2026-03-15`. Hand count: generator `GEN-N-1` at 100 MW once, plus `GEN-N-2` at 20 MW.
- `SYN-NORTH` queued requested MW is 50, status available, release `2026-04-01`. Hand count: request `Q-20` at 20 MW once, plus `Q-30` at 30 MW. The second `Q-20` row is a duplicate. `Q-AMB` (10 MW and 25 MW) is ambiguous and excluded. `Q-99` is withdrawn. `Q-77` is omitted.
- `SYN-NORTH` headroom is unobserved. The value is null. It is not 120 minus 50.
- `SYN-SOUTH` installed nameplate is missing. The value is null, not 0.
- `SYN-CENTRAL` installed nameplate is a reported 0 MW, status available.
- `SYN-CENTRAL` queued requested MW stays 40 and dated `2025-01-01`, status stale, because the age is past the 90-day threshold.
- `SYN-SOUTH` queued requested MW is incompatible because the source unit is MVA. The 999 MVA row is not published as a MW total.

Frozen as-of `2026-08-01` on the same files:

- `SYN-NORTH` queued requested MW remains 50 with release date `2026-04-01`, and the status changes to stale.
- `SYN-NORTH` installed nameplate remains 120, available, release `2026-03-15` (threshold 400 days).

A second region or fixture with no supported crosswalk displays unknown (status missing, reason `no_defensible_mapping`) and does not borrow another region's MW.

## Out of scope

Facility footprints, asset paths, outages, real-time topology, electrical flow, and any available-capacity or headroom estimate. Approximate rectangles in the demo are labeled synthetic. They are not administrative boundaries and not grid infrastructure.
