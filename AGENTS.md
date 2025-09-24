# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/`: CLI entrypoint hosting flag parsing and Fiber bootstrap.
- `internal/`: Go packages for env loading, database clients, hyperuser domain logic, events, etc. Keep new services under this tree using clear sub-package names.
- `test/hyperusers/`: Integration tests that exercise the running Fiber app against the configured databases.
- Root scripts (`RUNDEV`, `BUILD`, `TEST`, `RELEASE`) provide the standard build and release flow; prefer extending these instead of duplicating logic elsewhere.

## Build, Test, and Development Commands
- `./RUNDEV`: Starts the dev server with `--deployment dev` on port 8080 (uses repo `.env`).
- `./BUILD [--output DIR]`: Produces the release binary named after `VERSION`; default output is `bin/<version>`.
- `./TEST [--env-root PATH] [--app-version VERSION]`: Runs `go test ./...` with optional env overrides.
- `GOCACHE=$(pwd)/tmp/.gocache go build ./...`: Rebuilds the full module while keeping cache writable inside `tmp/`.
- `GOCACHE=$(pwd)/tmp/.gocache go test ./test/hyperusers`: Executes hyperuser integration tests; requires local Mongo/Redis populated with `testhyperuser` credentials.

## Coding Style & Naming Conventions
- Go code must remain `gofmt`-clean; run `gofmt -w` on touched files before committing.
- Use Go’s standard camelCase for variables/functions and UpperCamelCase for exported identifiers; package names stay all lowercase.
- Configuration constants should live in `internal/env` or a dedicated package so they can be reused by scripts/tests.

## Testing Guidelines
- Tests currently rely on Go’s standard testing package. Name files `<feature>_test.go` and keep helper code in package-level fixtures.
- Integration tests expect running backing services; seed MongoDB `hypervisor.hyperusers` with the `testhyperuser` account before `go test`.
- Add new suites under `test/<domain>/`; prefer table-driven tests for coverage and readability.

## Commit & Pull Request Guidelines
- Commit messages: short imperative subject (≤72 chars) with optional body explaining intent and side effects.
- Pull requests should link relevant issues, list manual/automated test runs (`./BUILD`, `./TEST`, targeted `go test`), and call out schema or environment changes.
- Include screenshots or logs when altering HTTP responses or operational scripts.

## Security & Configuration Tips
- Keep secrets such as `MONGO_URI` and `JWT_SECRET` in `.env`; never hard-code credentials in code or commits.
- When adding new scripts, source `.env` via `godotenv` or delegate to the existing env loader to avoid divergent configurations.
