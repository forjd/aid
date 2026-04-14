<div align="center">
  <h1>aid</h1>
  <p><strong>Local memory for coding agents and developers working inside Git repositories.</strong></p>
  <p>Capture notes, tasks, decisions, handoffs, and indexed commit history in a small Go CLI backed by SQLite.</p>
  <p>
    <code>local-first</code>
    <code>go</code>
    <code>sqlite</code>
    <code>git-aware</code>
    <code>agent-friendly</code>
  </p>
</div>

`aid` is a local-first command-line tool for preserving working context inside a repo. It gives humans and coding agents a lightweight memory layer that survives across sessions, stays close to the codebase, and can be queried without rebuilding context from scratch every time.

## Why `aid`

Agents are good at short bursts of execution and weak at continuity. They forget why a change was made, which task is still active, what a previous session learned, and which commits matter. `aid` exists to make repo work more resumable, more searchable, and less wasteful on tokens.

## Highlights

- Local-first storage with SQLite; no hosted service required.
- Built for real Git repositories, not a separate knowledge base.
- Predictable human-readable and machine-readable CLI output.
- Small, cross-platform Go binary with minimal runtime assumptions.
- Focused on continuity: notes, tasks, decisions, handoffs, and recall.

## Current Capabilities

| Area | Commands | What it covers |
| --- | --- | --- |
| Repo setup | `aid init`, `aid status` | Initialize repo state and inspect current memory health. |
| Working memory | `aid note`, `aid task`, `aid decide` | Record findings, active work, and engineering decisions. |
| Session continuity | `aid resume`, `aid handoff generate`, `aid handoff list` | Build compact summaries and persist handoffs between sessions. |
| Recall | `aid recall <query>` | Search notes, decisions, handoffs, and indexed commits together. |
| Git history | `aid history index`, `aid history search <query>` | Index and search local commit history with SQLite-backed retrieval. |
| Output modes | `--brief`, `--json`, `--verbose`, `--repo` | Support humans, scripts, and agent workflows. |

## Quickstart

### Requirements

- Go 1.26+
- A Git repository to work in

### Build

```bash
make build
./bin/aid --help
```

If you prefer to run without building a binary first:

```bash
go run ./cmd/aid --help
```

### Initialize and use it

```bash
./bin/aid init
./bin/aid status --brief
./bin/aid note add "Refresh token bug occurs after 401 retry"
./bin/aid task add "Tighten retry handling for expired sessions"
./bin/aid decide add "Store money as integer pence to avoid float drift"
./bin/aid history index
./bin/aid recall "refresh token"
./bin/aid resume --brief
./bin/aid handoff generate --brief
```

## Typical Workflow

1. Start a session with `aid resume --brief`.
2. Record only high-signal findings with `aid note add`.
3. Track meaningful units of work with `aid task add` and `aid task done`.
4. Save non-obvious engineering decisions with `aid decide add`.
5. Reindex and search commit history when Git context matters.
6. End with `aid handoff generate --brief`.

## Repository Layout

```text
cmd/aid/                CLI entrypoint
internal/app/           process bootstrapping and environment wiring
internal/cli/           command routing and help rendering
internal/config/        global and repo config loading
internal/git/           Git inspection helpers
internal/handoff/       handoff generation
internal/history/       commit indexing orchestration
internal/output/        human, brief, and JSON rendering
internal/resume/        resume bundle assembly
internal/search/        ranking and retrieval logic
internal/store/         storage interfaces
internal/store/sqlite/  SQLite implementation details
docs/spec/              product specs and longer-form design docs
skills/aid/             static skill package for compatible agents
```

## Project Status

`aid` is early, but the core local-memory workflow is already usable end to end. The current implementation includes real storage, config-driven defaults, FTS-backed recall across stored context, incremental history sync, and a practical resume/handoff loop.

The module path now matches the canonical repository path: `github.com/forjd/aid`.

## Documentation

- [MVP status](docs/mvp-status.md) for the current implementation tracker
- [MVP spec](docs/spec/mvp.md) for scope, goals, and non-goals
- [Architecture notes](docs/architecture.md) for package boundaries and design constraints
- [Agent skill package](skills/aid/SKILL.md) for agent usage guidance
- [Contributing guide](CONTRIBUTING.md) for development and pull request expectations
- [MIT license](LICENSE)

## Development

```bash
make fmt
make test
go run ./cmd/aid --help
```

If you are picking up implementation work, start with [docs/mvp-status.md](docs/mvp-status.md).
