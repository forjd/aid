# Repository Guidelines

## Project Structure & Module Organization
`cmd/aid/main.go` is the CLI entrypoint. Keep application code in `internal/`, with focused packages such as `cli/` for command wiring, `config/` for repo and global settings, `git/` for repository inspection, `history/`, `resume/`, `handoff/`, `search/`, `output/`, and `store/sqlite/` for persistence. Long-form docs live in `docs/`: use `docs/mvp-status.md` for current scope, `docs/architecture.md` for package boundaries, and `docs/spec/` for product intent. The checked-in agent skill is under `.agents/skills/aid/`. Do not commit generated artifacts from `bin/`, `dist/`, `.aid/`, or `coverage.out`.

## Build, Test, and Development Commands
Use the existing `Makefile` targets:

- `make build` builds `./bin/aid` from `./cmd/aid`.
- `make run` runs `go run ./cmd/aid --help` for a quick smoke test.
- `make fmt` applies `gofmt` to repository Go files.
- `make test` runs `go test ./...`.

For local iteration, `go run ./cmd/aid status --brief` is often faster than rebuilding the binary.

## Coding Style & Naming Conventions
Target Go `1.26` and let `gofmt` define formatting; do not add custom style rules. Follow standard Go naming: lowercase package names, `PascalCase` for exported identifiers, and `camelCase` for internal helpers. Keep commands and flags explicit, preserve stable human and JSON output, and avoid mixing rendering concerns into storage or domain logic. Prefer small, maintainable changes over clever abstractions.

## Testing Guidelines
Place tests next to the code they cover as `*_test.go`; current examples include `internal/cli/cli_test.go` and `internal/store/sqlite/store_test.go`. Add or update tests for behavior changes, especially around CLI output, repo-state handling, and SQLite-backed flows. Run `make test` before opening a PR. If you need a local coverage artifact, use `go test ./... -coverprofile=coverage.out`.

## Commit & Pull Request Guidelines
Commits that land on `main` should follow Conventional Commit semantics because `release-please` drives releases: `feat:` for minor releases, `fix:` for patches, and `type!:` or `BREAKING CHANGE:` for majors. Recent history follows this pattern, e.g. `feat: add release installer script` and `docs: add Homebrew distribution plan`. Keep PRs narrowly scoped, explain the problem being solved, note any CLI or output changes, update docs when behavior changes, and call out follow-up work that is intentionally out of scope. Do not create release tags manually.
