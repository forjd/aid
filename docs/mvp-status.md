# MVP Status

Last updated: 2026-04-15

This file is the repo's explicit implementation tracker.

Use it to answer:

- what is done
- what is partially done
- what is not started
- what the next session should work on

## Resume Here

If you are resuming work in a new session:

1. read this file first
2. skim [README.md](../README.md)
3. refer to [docs/spec/mvp.md](./spec/mvp.md) for scope and non-goals
4. run `make test-cover`
5. run `go run ./cmd/aid --help`

## Current Summary

The core local-memory workflow is implemented end to end. The coverage-hardening pass is now complete, CI enforces a whole-repo coverage floor, and the product direction is explicitly staying CLI-only for now rather than widening into a TUI, daemon, or sync service.

Working commands:

- `aid init`
- `aid status`
- `aid resume`
- `aid recall <query>`
- `aid note add`
- `aid note list`
- `aid task add`
- `aid task list`
- `aid task start`
- `aid task block`
- `aid task reopen`
- `aid task done`
- `aid decide add`
- `aid decide list`
- `aid handoff generate`
- `aid handoff list`
- `aid history index`
- `aid history search <query>`

Working global flags:

- `--json`
- `--brief`
- `--verbose`
- `--repo`

## Done

- Go project scaffold and package boundaries are in place.
- SQLite-backed local storage exists for repos, notes, tasks, decisions, handoffs, and indexed commits.
- `aid init` initialises repo state and writes `.aid/config.toml`.
- Notes can be added and listed.
- Tasks can be added, listed, and completed.
- Tasks can now also be moved directly to `in_progress`, `blocked`, and back to `open`.
- Decisions can be added and listed.
- `aid status` reports repo state and counts.
- `aid resume` builds a compact working summary with active-task inference.
- `aid handoff generate` saves a persistent handoff snapshot.
- `aid handoff list` reads saved handoffs.
- `aid history index` stores commit metadata in SQLite.
- `aid history search` searches indexed commits with SQLite FTS ranking.
- `aid recall` searches notes, decisions, handoffs, and indexed commits together with SQLite FTS ranking across all stored context types.
- Human-readable output, `--brief`, `--verbose`, and `--json` are implemented for the working commands.
- The repo includes a static skill package at [.agents/skills/aid/SKILL.md](../.agents/skills/aid/SKILL.md).
- Tests cover the main end-to-end command flows.
- `aid resume` now carries richer next-action heuristics, open questions, and the latest saved handoff when available.
- `aid handoff generate` now includes open questions and more deliberate next-action synthesis.
- `aid resume` and `aid handoff generate` now reuse indexed commits when available instead of relying only on live Git reads.
- `aid resume` and `aid handoff generate` now carry forward prior handoff questions and next actions when current context is otherwise thin.
- Repo config is now used for default output mode selection and history indexing ignore paths.
- `aid history index` now performs incremental sync against stored commits instead of clearing and replacing the index each run.
- `aid history index` now reconciles against reachable commit SHAs first and only fetches full metadata for newly discovered commits.
- SQLite migrations now use an explicit schema version.
- The Go module path is now the canonical `github.com/forjd/aid`.
- The repo now explicitly stays FTS-first; optional embeddings are deferred until real recall gaps justify the added complexity.
- FTS-only recall has now been validated in larger real repos; exact and near-exact lexical queries perform well enough to keep embeddings deferred for now.
- Concurrent `aid recall` runs against the same repo no longer trip transient SQLite `database is locked` failures during store open.
- Direct tests now cover `internal/output`, `internal/resume`, `internal/handoff`, `internal/search`, `internal/config`, and `internal/git`.
- `internal/cli` and `internal/store/sqlite` package-local coverage now meet the `80%` floor.
- `make test-cover` is now the standard whole-repo coverage command for local work, and CI fails if whole-repo coverage drops below `80%`.
- Post-MVP direction is now explicitly CLI-only for the near term; TUI, background automation, and multi-user sync remain deferred.

## Partial

- None at the moment.

## Not Started

- Any TUI, background daemon, cloud sync, or team collaboration work.

## MVP Acceptance Checklist

- [x] initialise `aid` in a Git repo
- [x] add and list notes
- [x] add, list, and complete tasks
- [x] add and list decisions
- [x] generate a useful handoff
- [x] index Git history
- [x] search indexed history
- [x] run `aid resume` and get a compact, useful context summary
- [x] use `--json` output for scripting or agents
- [x] include a static `SKILL.md` skill package in the repo

## Recommended Next Work

1. Improve lexical normalisation and phrasing tolerance inside the existing recall flow, especially gaps like `town house` vs `townhouse` and `optimise` vs `speed up`.
2. Keep tightening CLI ergonomics, recall ranking, and summary quality without expanding the product surface beyond the current repo-local CLI model.

## Open Notes

- The current implementation already satisfies the practical MVP command surface.
- Remaining work should stay within the current CLI-first product shape unless real usage shows sustained pressure for something larger.
- Recent validation in larger repos suggests the main recall misses are tokenisation and phrasing gaps such as `town house` vs `townhouse` or `optimise` vs `speed up`, rather than a broad need for embeddings.
- Coverage guardrails are now in place, so feature work should preserve the shared `make test-cover` workflow instead of inventing new local-only measurements.
- Future sessions should treat this file as the source of truth for implementation status.
