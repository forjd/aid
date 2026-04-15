# Architecture

## Intent

Keep `aid` small, local-first, and boring.

The repo should optimise for:

- a clear CLI surface
- strict separation between rendering and business logic
- SQLite-backed local storage
- Git-aware retrieval and indexing
- a package layout that can grow without becoming framework-shaped

## Package Boundaries

- `cmd/aid`: process entrypoint only
- `internal/cli`: command routing, argument handling, help rendering
- `internal/app`: bootstrapping, repo resolution, app-data paths, environment wiring
- `internal/config`: global config and `.aid/config.toml` loading
- `internal/store`: storage interfaces and repository-facing persistence contracts
- `internal/store/sqlite`: schema management and SQLite-backed persistence
- `internal/git`: Git repo inspection, branch lookup, and commit enumeration
- `internal/history`: commit indexing workflow orchestration
- `internal/search`: ranking and retrieval across memory objects
- `internal/resume`: assembly of the compact resume bundle
- `internal/handoff`: handoff generation and persistence
- `internal/output`: human, brief, and JSON renderers

## Dependency Direction

The dependency rule is simple:

- `cmd/aid` imports `internal/cli`
- `internal/cli` coordinates application packages
- storage and Git packages return plain data, not formatted terminal strings
- `internal/output` renders plain data and output-specific DTOs into terminal or JSON output

Avoid turning `aid` into a library-first design. This is an application, not a reusable SDK, so `internal/` should be the default.

## Why Not `pkg/`

There is no external API to stabilise yet.

Adding `pkg/` now would suggest a reusable library boundary that the project does not actually need. If a genuine public package emerges later, it can be carved out deliberately.

## Suggested Delivery Order

1. Define domain models and the initial SQLite schema.
2. Implement `init`, `note`, `task`, and `decide`.
3. Add shared output renderers for default, brief, and JSON modes.
4. Implement `resume` and `handoff generate`.
5. Add commit indexing and history search.
6. Tune ranking, help text, and skill-package guidance.

## README Policy

Use `README.md` as the contributor-facing homepage.

Put longer, decision-heavy product material in `docs/spec/` so the entrypoint stays easy to scan.
