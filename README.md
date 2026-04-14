# aid

`aid` is a local memory CLI for coding agents and developers working inside Git repositories.

It is being built in Go as a small, cross-platform tool with SQLite-backed local storage, strong help output, and predictable human and machine-readable command surfaces.

## Status

This repository now has a Go-first scaffold:

- a compilable CLI entrypoint
- a command tree and help surface
- package boundaries for storage, Git, search, resume, handoff, and output
- a static skill package location for agents

The implementation is intentionally shallow at this stage. Commands exist as stubs so the repo shape, docs, and build flow are settled before storage and workflows land.

## Why Go

Go is the right fit for `aid` because it keeps distribution and runtime simple:

- single-binary delivery
- straightforward filesystem and subprocess work
- predictable cross-platform behaviour
- good fit for SQLite-backed local tools
- aligned with the project's "boring technology" constraint

## Repository Layout

```text
cmd/aid/             CLI entrypoint
internal/cli/        command routing and help rendering
internal/app/        process bootstrapping and environment wiring
internal/config/     global and repo config loading
internal/git/        Git inspection helpers
internal/history/    commit indexing orchestration
internal/handoff/    handoff generation
internal/output/     human, brief, and JSON rendering
internal/resume/     resume bundle assembly
internal/search/     ranking and retrieval logic
internal/store/      storage interfaces
internal/store/sqlite/ SQLite implementation details
docs/spec/           product specs and longer-form design docs
skills/aid/          static skill package for compatible agents
```

## Quickstart

```bash
make build
go run ./cmd/aid --help
go test ./...
```

The Go module currently uses `module aid` as a local-safe placeholder. Rename it once the repository has a canonical remote path.

## Documentation

- [MVP spec](docs/spec/mvp.md)
- [Architecture notes](docs/architecture.md)
- [Agent skill package](skills/aid/SKILL.md)

## What To Do With The README

Keep `README.md` short and repo-facing.

Use it for:

- the one-paragraph product description
- current implementation status
- the repo layout
- quickstart commands
- links to the deeper docs

Do not use it as the full product spec. That content belongs in `docs/spec/mvp.md`, where it can grow without turning the repo homepage into a wall of text.


If the MVP works, `aid` becomes the first command an agent runs when entering a repository, and the last command it runs before leaving.

The next sensible step is to turn this into a polished `README.md` plus a first draft of a static `SKILL.md` package.
