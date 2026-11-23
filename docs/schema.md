# GridAtlas schemas

Input schema: `gridatlas.input.v1`

Output schema: `gridatlas.atlas.v1`

Quantities in v1 are non-negative integers. The synthetic fixtures use whole MW, and the scorer compares them exactly. A fractional MW string is rejected rather than rounded.

## Input directory

| File | Required columns |
| --- | --- |
| `regions.csv` | `region_id`, `name` |
| `crosswalk.csv` | `source_region`, `atlas_region`, `mapping_method`, `supported` |
| `manifest.csv` | `product`, `source_url`, `release_date`, `reporting_period`, `unit`, `region`, `mapping_method`, `content_sha256`, `snapshot_file`, `freshness_threshold_days` |
| `regions.geojson` | FeatureCollection of polygons. Each feature sets `synthetic` to true and a `boundary_note` that says the shape is synthetic. |
| snapshot CSV | `record_id`, `request_id`, `technology`, `quantity`, `unit`, `basis`, `row_status` |

Products are `installed_nameplate` and `queued_requested`. One manifest row per region per product. A missing row is a missing cell, not a zero.

`supported` is `true` or `false`. `explicit_region` is the only join the sample marks supported. `state_sum` in the unsupported fixture is explicitly false.

`basis` is `nameplate_mw` or `requested_mw`. `row_status` is `operating`, `active`, `withdrawn`, `omitted`, or `ambiguous`.

`source_url` is `fixture://...` for authored rows or `https://...` for a later user import. `content_sha256` is the lowercase SHA-256 of the snapshot file bytes.

## Bounds

- Each input file is at most 1 MiB.
- A snapshot has at most 10,000 data rows, and a bundle has at most 10,000 snapshot rows in total.
- A bundle has at most 64 regions.
- A quantity is an integer from 0 through 10,000,000.
- Freshness thresholds are integers from 0 through 3,650 days.
- Snapshot paths are relative and must stay inside the input directory.

## Error categories

Fatal input errors use exit code 2 and the prefix `input invalid (<category>)`:

| Category | When |
| --- | --- |
| `schema` | Missing column, bad date, unknown product, non-integer quantity, geometry that is not labeled synthetic |
| `bounds` | File size, row count, quantity, or threshold outside the limits above |
| `checksum` | Snapshot bytes do not match `content_sha256` |
| `path` | Absolute path, `..`, or an output directory inside the input directory |
| `io` | The file cannot be read |

A bundle that loads can still contain incompatible cells. Those are findings, not fatal errors.

## Output

`atlas.json` lists regions in `region_id` order. Each region has `installed_nameplate_mw`, `queued_requested_mw`, and `headroom_mw`.

Cell status is `available`, `stale`, `missing`, or `incompatible`. `display_status` is `unknown` when the reason is `no_defensible_mapping`. Headroom status is always `unobserved` and `value_mw` is null.

`zero` is true only when the source included a row and the sum is 0.

A cell is stale when `as_of - release_date` in whole UTC days is greater than that row's `freshness_threshold_days`. The release date and value stay in the cell.

`summary.md` repeats the same statuses, values, definitions, snapshot paths, and checksums.
