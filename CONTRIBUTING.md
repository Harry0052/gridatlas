# Contributing

GridAtlas is a local coverage atlas. Changes should keep installed nameplate, queued requests, and headroom as separate fields.

## Setup

Go 1.22 or newer and Node.js 22 are enough. From a clean checkout:

```bash
npm ci
bash scripts/ci.sh
```

`bash scripts/ci.sh` installs dependencies, typechecks the map, runs the Go tests with the race detector, and runs the fixture demo.

## Fixtures

Demo rows in `testdata/fixtures` are synthetic and authored by hand. Do not add a real generator inventory, interconnection queue extract, facility coordinate, or line route. A new real-data importer has to land in the CSV contract in `docs/schema.md` and leave the upstream file outside the repository.

## Pull requests

Run `bash scripts/ci.sh` before opening a pull request. The GitHub Actions workflow runs that script with read-only permissions and no secrets.
