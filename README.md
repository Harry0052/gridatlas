# GridAtlas

GridAtlas shows where coarse public planning figures for regional grid capacity are available, missing, stale, or on an incompatible basis. It keeps two products separate:

- operating generator **nameplate MW**
- distinct active interconnection requests' **requested MW**

Electrical headroom is **unobserved**. The program does not subtract the queue from installed capacity, and it does not draw facilities, lines, or flows.

The demo uses three synthetic regions. It does not measure a real grid.

## Setup

Install Go 1.22 or newer and Node.js 22. From a clean checkout:

```bash
npm ci
```

No API key is required. Scoring a bundle does not use the network.

## Demo

From the repository root:

```bash
npm run demo
```

That compiles the map and writes:

| File | What it is |
| --- | --- |
| `dist/demo/summary.md` | Human-readable report |
| `dist/demo/atlas.json` | Machine-readable report (`gridatlas.atlas.v1`) |
| `dist/demo/report.html` | Map, table, and the same export buttons |
| `dist/demo/regions.geojson` | The synthetic rectangles used by the map |

Expected result for as-of **2026-06-01**:

- `SYN-NORTH` installed nameplate **120 MW**, available, released 2026-03-15
- `SYN-NORTH` queued requested **50 MW**, available, released 2026-04-01
- `SYN-NORTH` headroom **unobserved**
- `SYN-CENTRAL` installed nameplate **0 MW (reported zero)**, available
- `SYN-CENTRAL` queued requested **40 MW**, stale, released 2025-01-01
- `SYN-SOUTH` installed nameplate **missing** (not zero)
- `SYN-SOUTH` queued requested **incompatible** because the source unit is MVA

Open the map from the report directory:

```bash
go run ./cmd/gridatlas serve --dir dist/demo
```

Then open `http://127.0.0.1:8787/report.html`. The server binds to loopback only. Choose a metric, select a region from the map or the table, and use **Download JSON** or **Download summary**. The trace names the snapshot file and the SHA-256 in the manifest.

A second as-of date shows the queue cell age without dropping its value:

```bash
go run ./cmd/gridatlas run --input testdata/fixtures/sample --as-of 2026-08-01 --out dist/stale
```

`SYN-NORTH` queued requested MW stays **50** with release date **2026-04-01**, and the status becomes **stale**.

## Checks

```bash
bash scripts/ci.sh
```

The script installs dependencies, typechecks the map, runs `go test -race`, builds `dist/gridatlas`, runs the demo, and checks that a malformed manifest exits 2.

Useful commands on their own:

```bash
go run ./cmd/gridatlas help
go run ./cmd/gridatlas validate --input testdata/fixtures/sample
go test ./...
npm test
```

`go run` prints a program's non-zero status and then exits 1. Use the built binary when you need the exit code itself:

```bash
go build -o dist/gridatlas ./cmd/gridatlas
./dist/gridatlas run --input testdata/fixtures/invalid --as-of 2026-06-01 --out dist/invalid-out
```

That command exits **2** and prints `missing required column "unit"`.

## What a cell means

| Status | Meaning |
| --- | --- |
| available | The source joined on a supported crosswalk, the unit is MW, and the release is inside its threshold. A value of 0 is a reported zero. |
| stale | The same value and release date, with the as-of date past that dataset's threshold. |
| missing | No manifest row, or no included rows. Not a zero. |
| unknown | The region has no defensible crosswalk entry. Stored as status `missing` with reason `no_defensible_mapping`. |
| incompatible | The unit is not MW, the basis does not match the product, or the crosswalk marks the join unsupported. |
| unobserved | Headroom. Always. |

Definitions, thresholds, and the versioned columns are in [docs/schema.md](docs/schema.md). The acceptance arithmetic is in [testdata/oracle/sample.txt](testdata/oracle/sample.txt).

## Real data

The demo does not download EIA-860M or Berkeley Lab Queued Up. Fixture terms are in [NOTICE](NOTICE) and [testdata/fixtures/sample/README.txt](testdata/fixtures/sample/README.txt).

To score a dataset you already have the rights to, aggregate it yourself to the coarse region and write the CSV contract in [docs/schema.md](docs/schema.md). Keep the upstream file outside this repository. A state-to-region sum is incompatible unless the crosswalk marks that method `supported` and you accept the overlap.

There is no separate hardware or model demo.

## License and contributing

MIT. See [LICENSE](LICENSE), [CONTRIBUTING.md](CONTRIBUTING.md), and [SECURITY.md](SECURITY.md).
