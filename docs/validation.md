# Validation

Access date: 2026-11-23. The checks below were run on the synthetic fixtures in this repository. They are not a measurement of a physical grid, and they are not a check of an EIA or Berkeley Lab workbook.

## Environment

- Host: Linux 6.12.113+ x86_64, 4 CPUs
- Go: go1.22.2 linux/amd64
- Node: v22.23.4
- TypeScript: 5.6.3, development only
- Go module dependencies: none. There is no `go.sum` because the program uses the standard library only.
- npm audit after `npm ci`: 0 vulnerabilities

## Acceptance metric

Tolerance is exact integer MW. Source is the hand count in `testdata/oracle/sample.txt`.

`go test ./...` compares every cell with that oracle, the freshness boundary oracle, the unknown-mapping oracle, and the unsupported-join oracle. The run on 2026-11-23 passed.

Observed demo at as-of 2026-06-01 (`dist/demo/summary.md`):

| Cell | Status | Value | Release | Age |
| --- | --- | --- | --- | --- |
| SYN-NORTH installed nameplate | available | 120 MW | 2026-03-15 | 78 days |
| SYN-NORTH queued requested | available | 50 MW | 2026-04-01 | 61 days |
| SYN-NORTH headroom | unobserved | unobserved |  |  |
| SYN-CENTRAL installed nameplate | available | 0 MW (reported zero) | 2026-05-01 | 31 days |
| SYN-CENTRAL queued requested | stale | 40 MW | 2025-01-01 | 516 days |
| SYN-CENTRAL headroom | unobserved | unobserved |  |  |
| SYN-SOUTH installed nameplate | missing | missing |  |  |
| SYN-SOUTH queued requested | incompatible | incompatible | 2026-05-01 | 31 days |
| SYN-SOUTH headroom | unobserved | unobserved |  |  |

The south queue trace keeps the rejected row as `999 MVA`, not as a MW total. The summary does not contain `70 MW` or `999 MW`.

A second run, `./dist/gridatlas run --input testdata/fixtures/sample --as-of 2026-08-01 --out dist/stale`, left SYN-NORTH queued at 50 MW with release date 2026-04-01 and changed the status to stale (age 122 days). Installed nameplate stayed 120 MW, available, age 139 days, threshold 400 days.

The boundary fixture is fresh at age 30 (as-of 2026-01-31, threshold 30) and stale at age 31 (as-of 2026-02-01) with the value still 10 MW.

## Domain safeguard

Every metric section in the summary prints the definition, and a dated cell also prints the release date. Headroom is a separate section whose value is unobserved. Missing installed capacity for SYN-SOUTH is the word missing, not 0. The map geometry is a labeled synthetic rectangle. The report does not draw facilities or lines.

Imported files are capped at 1 MiB and 10,000 rows. Snapshot paths must stay inside the input directory. Symlinks are rejected. The HTML report JSON-escapes `<` so a source string cannot close the script tag (`TestMarshalEscapesMarkup`). The map writes text with `textContent`.

`./dist/gridatlas run --input testdata/fixtures/invalid --as-of 2026-06-01 --out dist/invalid-out` exits 2 and prints:

```text
input invalid (schema): manifest.csv: missing required column "unit"
```

`go run` prints that status and then exits 1. The built binary returns 2. The README says so.

## Workflow check

The page at `dist/demo/report.html` was opened at `http://127.0.0.1:8787/report.html`.

- The installed metric announced `Installed nameplate MW. 2 available, 1 missing.` SYN-SOUTH was missing. SYN-CENTRAL was `0 MW (reported zero)`.
- Choosing queued requested MW announced `Queued requested MW. 1 stale, 1 available, 1 incompatible.` SYN-NORTH was 50 MW. SYN-SOUTH was incompatible.
- ArrowRight from that radio moved to unobserved headroom. The live region, which is `aria-live="polite"`, read `Unobserved headroom. 3 unobserved.` The focused control's outline was solid, about 2px, color rgb(11, 58, 74).
- Enter on the north map button announced `Queued requested MW, North Planning Region (synthetic), 50 MW, available, released 2026-04-01.` The trace named `snapshots/north-queue.csv`, the manifest SHA-256, included Q-20 once and Q-30, and marked the second Q-20 row duplicate.
- At 375px width the layout was one column and the document did not scroll horizontally (`scrollWidth` 360). At 1100px width the layout was two columns.
- The only console error was a missing favicon. It does not affect the report.

## Performance

Budget: the sample demo command finishes in under 1 second on this host. Observed `./dist/gridatlas demo` wall time: 6.2 ms. Sample fixture size: 4,256 bytes. Outputs: `atlas.json` 13,225 bytes, `summary.md` 6,068 bytes, `report.html` 39,190 bytes.

## Dependency and license check

The Go program has no third-party modules. The map's only package is TypeScript 5.6.3, used at build time, MIT licensed. `npm audit` reported 0 vulnerabilities. Fixture rows are original. EIA and Queued Up files are not in the tree. See `NOTICE`.

The race detector (`go test -race ./...`) passed. This is Go, so an address or undefined-behavior sanitizer for C is not part of the toolchain. The race detector is the dynamic check the toolchain provides. `CGO` is used only by that detector.

## Commands

```text
go test ./...
bash scripts/ci.sh
```

`bash scripts/ci.sh` on 2026-11-23 printed `ci ok` after `npm ci`, the typecheck, the Node test, `go vet`, `go test -race`, the demo, and the invalid-input exit check.

## Residual limitations

- Real EIA-860M and Queued Up files are not scored here. Redistribution of the Queued Up workbook is still unresolved, and no planning-region crosswalk was validated.
- Headroom stays unobserved on purpose. The atlas will not fill it in from the other two metrics.
- `go run` does not pass the program's exit code through. Use `dist/gridatlas` when the exit code matters.
