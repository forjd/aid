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

The core local-memory workflow is implemented end to end.

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
- `aid history search` searches indexed commits.
- `aid recall` searches notes, decisions, handoffs, and indexed commits together.
- Human-readable output, `--brief`, and `--json` are implemented for the working commands.
- The repo includes a static skill package at [skills/aid/SKILL.md](../skills/aid/SKILL.md).
- Tests cover the main end-to-end command flows.

## Partial

- `--verbose` is parsed globally but does not yet have distinct richer renderers.
- History search works, but it currently uses simple SQL `LIKE` matching rather than SQLite FTS.
- Recall ranking is basic branch-aware filtering, not tuned relevance scoring.
- Handoff summaries are useful, but still heuristic and fairly simple.
- Repo config is created, but only lightly used after init.
- The Go module path is still the local placeholder `module aid`.

## Not Started

- SQLite FTS-backed history and recall ranking.
- Optional embeddings / semantic search.
- Incremental history indexing instead of replacing the full commit index each run.
- Richer handoff generation with open questions and more deliberate next-step synthesis.
- Distinct `--verbose` output mode.
- Better schema migration/versioning strategy.
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

1. Replace commit search with SQLite FTS and reuse that ranking in `recall`.
2. Make `--verbose` materially different from the default human-readable output.
3. Improve `resume` and handoff ranking so active work and suggested next steps are more reliable.
4. Move from full history reindexing to incremental indexing.
5. Rename the Go module once the canonical repository path is known.

## Open Notes

- The current implementation already satisfies the practical MVP command surface.
- Remaining work is mostly quality, ranking, and polish rather than missing core commands.
- Future sessions should treat this file as the source of truth for implementation status.

