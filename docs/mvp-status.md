# MVP Status

Last updated: 2026-04-14

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

The core local-memory workflow is implemented end to end, and the main post-MVP quality gaps have been closed.

Working commands:

- `aid init`
- `aid status`
- `aid resume`
- `aid recall <query>`
- `aid note add`
- `aid note list`
- `aid task add`
- `aid task list`
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
- Decisions can be added and listed.
- `aid status` reports repo state and counts.
- `aid resume` builds a compact working summary with active-task inference.
- `aid handoff generate` saves a persistent handoff snapshot.
- `aid handoff list` reads saved handoffs.
- `aid history index` stores commit metadata in SQLite.
- `aid history search` searches indexed commits with SQLite FTS ranking.
- `aid recall` searches notes, decisions, handoffs, and indexed commits together with SQLite FTS ranking across all stored context types.
- Human-readable output, `--brief`, `--verbose`, and `--json` are implemented for the working commands.
- The repo includes a static skill package at [skills/aid/SKILL.md](../skills/aid/SKILL.md).
- Tests cover the main end-to-end command flows.
- `aid resume` now carries richer next-action heuristics, open questions, and the latest saved handoff when available.
- `aid handoff generate` now includes open questions and more deliberate next-action synthesis.
- Repo config is now used for default output mode selection and history indexing ignore paths.
- `aid history index` now performs incremental sync against stored commits instead of clearing and replacing the index each run.
- SQLite migrations now use an explicit schema version.
- The Go module path is now the canonical `github.com/forjd/aid`.

## Partial

- History indexing still reads full Git history before reconciling the local index, even though the SQLite write path is now incremental.

## Not Started

- Optional embeddings / semantic search.
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

1. Decide whether optional embeddings / semantic search are worth adding beyond the new FTS-based recall.
2. Reduce `history index` Git read cost so it can avoid scanning the full history before syncing.
3. Expand task lifecycle commands beyond `task done` so agents can set `in_progress` and `blocked` directly.
4. Explore whether resume and handoff synthesis should start using indexed history and prior handoffs more aggressively.

## Open Notes

- The current implementation already satisfies the practical MVP command surface.
- Remaining work is now mostly optional search depth and future product expansion rather than missing or weak core commands.
- Future sessions should treat this file as the source of truth for implementation status.
