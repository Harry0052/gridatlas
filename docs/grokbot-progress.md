# Grokbot progress

## Index

| Phase | Status |
| --- | --- |
| P01 | PASS |
| P02 | PASS |
| P03 | PASS |
| P04 | PASS |
| P05 | PASS |
| P06 | PASS |
| P07 | not started |

## P01 — Research overlap and confirm the MVP

**Phase and status:** P01 PASS

**Files changed:**

- `docs/overlap.md`
- `docs/project-contract.md`
- `docs/grokbot-progress.md`

**Commands and results:**

- GitHub repository search `interconnection queue coverage atlas capacity` on 2026-11-23: `total_count` 0.
- GitHub repository search `queued up berkeley lab interconnection`: one hit, `keanuhea/interconnection-queue-analysis` (pushed 2026-05-18, license field null). Metadata only.
- `gridstatus/gridstatus`: BSD-3-Clause, pushed 2026-11-20, 456 stars. PyPI `gridstatus` version 0.37.0, OSI BSD license classifier.
- `catalyst-cooperative/pudl`: MIT, pushed 2026-11-22, 622 stars. Docs title "PUDL 2026.11.0".
- npm search `gridatlas`: `No matches found for "gridatlas"`.
- Local probe (Python 3.12, no third-party packages):

```text
gridstatus import: unavailable: ModuleNotFoundError No module named 'gridstatus'
deduped_queued_mw 50
ambiguous_requests 1
naive_headroom_mw 15
acceptance_headroom unobserved
naive_matches_acceptance False
```

- Primary pages read on 2026-11-23: `https://emp.lbl.gov/queues`, Queued Up 2026 and 2024 edition decks, `https://www.eia.gov/electricity/data/eia860m/`, `https://www.eia.gov/about/copyrights_reuse.php`, `https://opensource.gridstatus.io/en/latest/`, `https://github.com/gridstatus/gridstatus`, `https://github.com/catalyst-cooperative/pudl`, GridCapacity capacity-model and freshness docs, UK Grid Atlas README, GB Grid Queue Map README.

**Artifact paths:**

- `docs/overlap.md`
- `docs/project-contract.md`

**Open issues:**

- Queued Up workbook redistribution terms are unresolved. The workbook is not part of the demo.
- An active-status ISO rollup was not reproduced from the workbook. Editions do not share a stable coverage universe (44 versus 50 non-ISO areas).
- No EIA planning-region crosswalk was computed. Real regional sums stay out of scope until a supported crosswalk exists.

**Next phase:** P02

The evidence supports a concrete MVP the existing tools do not already score. Synthetic fixtures have a clear provenance path (authored in this repository). EIA-860M remains an optional later import under public-domain terms and is not required to test the acceptance case.

## P02 — Create the repository and data contracts

**Phase and status:** P02 PASS

**Files changed:**

- `go.mod`, `cmd/gridatlas`, `internal/ingest`, `internal/model`
- `testdata/fixtures/sample`, `testdata/fixtures/invalid`, `docs/schema.md`
- `README.md` install and test commands

**Commands and results:**

- Toolchain: `go version go1.22.2 linux/amd64`. No third-party Go modules.
- `go run ./cmd/gridatlas validate --input testdata/fixtures/sample`

```text
valid gridatlas.input.v1
regions: 3
manifest_rows: 5
```

- Built binary on the invalid fixture, exit 2:

```text
input invalid (schema): manifest.csv: missing required column "unit"
```

**Artifact paths:**

- `docs/schema.md`
- `testdata/fixtures/sample/README.txt`
- `testdata/fixtures/invalid/manifest.csv`

**Open issues:** none for this gate. Checksums, row limits, and path checks are covered by `go test ./internal/ingest`.

**Next phase:** P03

## P03 — Build the working core

**Phase and status:** P03 PASS

**Files changed:**

- `internal/engine`, `internal/export`
- `testdata/oracle/sample.txt`, `boundary.txt`, `unknown.txt`, `unsupported.txt`

**Commands and results:**

- `go test ./internal/engine` passed. The test loads the fixtures and compares each cell with the hand-written oracle.
- Demo as-of 2026-06-01 wrote `dist/demo/atlas.json` and `dist/demo/summary.md`. SYN-NORTH installed is 120 MW available. SYN-NORTH queued is 50 MW available. Headroom is unobserved. SYN-SOUTH installed is missing.
- `./dist/gridatlas run --input testdata/fixtures/sample --as-of 2026-08-01 --out dist/stale` kept queued at 50 MW with release date 2026-04-01 and set the status to stale (age 122). Installed stayed 120 MW available (age 139, threshold 400).

**Artifact paths:**

- `dist/demo/atlas.json`
- `dist/demo/summary.md`
- `dist/stale/atlas.json`
- `testdata/oracle/sample.txt`

**Open issues:** none for this gate. Repeated request IDs and the 10 MW / 25 MW Q-AMB pair are excluded from the 50 MW sum.

**Next phase:** P04

## P04 — Ship the user workflow and reports

**Phase and status:** P04 PASS

**Files changed:**

- `web/src/atlas.ts`, `web/report.template.html`, `web/test/atlas.test.ts`
- `cmd/gridatlas` `demo` and `serve`

**Commands and results:**

- `npm run demo` writes the JSON report, the summary, and `dist/demo/report.html`.
- `npm test` passed the filter test: installed 120, queued 50, and headroom null stay distinct.
- Served `http://127.0.0.1:8787/report.html` and exercised it in a browser.
  - Installed view: 2 available, 1 missing, including a reported zero.
  - Queued view: 1 stale, 1 available (50 MW), 1 incompatible.
  - ArrowRight announced `Unobserved headroom. 3 unobserved.` Focus outline was solid, color rgb(11, 58, 74).
  - Enter on the north region announced the 50 MW cell with release date 2026-04-01. The trace named `snapshots/north-queue.csv` and the manifest SHA-256.
  - 375px width: one column, no horizontal page scroll. 1100px width: two columns.
- Invalid input through the normal command exits 2 with `missing required column "unit"`.

**Artifact paths:**

- `dist/demo/report.html`
- `dist/demo/summary.md`

**Open issues:** the page requests `/favicon.ico` and the server returns 404. The report itself loads.

**Next phase:** P05

## P05 — Verify correctness, security, and performance

**Phase and status:** P05 PASS

**Files changed:**

- `docs/validation.md`
- symlink refusal in `internal/ingest` and `internal/export`

**Commands and results:**

- `go test ./...` passed, including oracle, escape, path, checksum, and symlink tests.
- `go test -race ./...` passed inside `bash scripts/ci.sh`.
- `npm audit` after `npm ci`: 0 vulnerabilities.
- `./dist/gridatlas demo` measured 6.2 ms. Budget is 1 second. Sample fixture is 4,256 bytes.
- Address and undefined-behavior sanitizers are for C and C++. This program is Go. The race detector is the dynamic check that ran.

**Artifact paths:**

- `docs/validation.md`

**Open issues:** real EIA and Queued Up files remain out of the demo, as decided in P01. Headroom is unobserved by design.

**Next phase:** P06

## P06 — Prepare a reproducible open-source release

**Phase and status:** P06 PASS

**Files changed:**

- `README.md`, `SECURITY.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `NOTICE`
- `scripts/ci.sh`, `.github/workflows/ci.yml`, `package-lock.json`

**Commands and results:**

- A clean copy at `/tmp/gridatlas-clean` (no `node_modules`, no `dist`, no `web/dist`) ran `bash scripts/ci.sh` before the final symlink guard. It printed `ci ok`.
- After the symlink guard, `bash scripts/ci.sh` from `/workspace` printed `ci ok` again: `npm ci`, typecheck, Node test, `go vet`, `go test -race`, demo, invalid exit 2, and the summary check.
- The workflow uses read-only `contents` permission and pins actions to commit SHAs. It does not use secrets.
- Hosted GitHub Actions is the P07 check.

**Artifact paths:**

- `.github/workflows/ci.yml`
- `dist/demo/summary.md` from the CI script

**Open issues:** none that block publishing. The tag and the hosted run happen in P07.

**Next phase:** P07
