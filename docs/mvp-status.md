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
4. run `go test ./...`
5. run `go run ./cmd/aid --help`

## Current Summary

The core local-memory workflow is implemented end to end, and the previously listed follow-up gaps around task state control, history indexing cost, resume / handoff reuse, and real-repo FTS validation have now been closed.

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

1. Start the coverage-hardening pass in [docs/plans/code-coverage-improvement.md](./plans/code-coverage-improvement.md), beginning with direct tests for `internal/output`, `internal/resume`, and `internal/handoff`.
2. Decide whether post-MVP expansion should go toward a TUI, background automation, or multi-user sync instead of keeping the tool intentionally CLI-only.

## Open Notes

- The current implementation already satisfies the practical MVP command surface.
- Remaining work is now product-expansion work rather than missing core command behavior.
- Recent validation in larger repos suggests the main recall misses are tokenisation and phrasing gaps such as `town house` vs `townhouse` or `optimise` vs `speed up`, rather than a broad need for embeddings.
- Future sessions should treat this file as the source of truth for implementation status.
