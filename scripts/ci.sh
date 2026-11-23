#!/usr/bin/env bash
# Same checks as GitHub Actions. Run from anywhere; the script cds to the repo root.
set -euo pipefail
cd "$(dirname "$0")/.."

npm ci
npm run build
npm run typecheck
npm test
go vet ./...
go test -race ./...
mkdir -p dist
go build -o dist/gridatlas ./cmd/gridatlas

./dist/gridatlas validate --input testdata/fixtures/sample
./dist/gridatlas demo

set +e
./dist/gridatlas run --input testdata/fixtures/invalid --as-of 2026-06-01 --out dist/invalid-out
code=$?
set -e
test "$code" -eq 2

python3 - << 'PY'
from pathlib import Path
text = Path("dist/demo/summary.md").read_text()
required = ["120 MW", "50 MW", "unobserved", "0 MW (reported zero)", "Value: missing", "Value: incompatible", "999 MVA rejected_unit"]
missing = [item for item in required if item not in text]
if missing:
    raise SystemExit("demo summary missing: " + ", ".join(missing))
if "70 MW" in text or "999 MW" in text:
    raise SystemExit("demo summary published a rejected total")
print("demo summary matches the acceptance case")
PY
echo "ci ok"
