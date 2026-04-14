# aid MVP

## Overview

`aid` is a local CLI tool that gives AI coding agents a durable working memory inside a Git repository.

It is designed to solve a simple problem: agents repeatedly lose context between sessions. They forget what they were doing, why certain decisions were made, which commits matter, and what should happen next.

`aid` provides a lightweight memory layer for both humans and agents by combining:

- scratch notes
- task tracking
- engineering decisions
- handoff summaries
- searchable Git history

The MVP should stay narrow. This is not an agent framework, a chat interface, or a general-purpose knowledge base. It is a repo-aware memory and recall tool.

---

## Problem

AI agents are good at short bursts of work, but weak at continuity.

Common failure modes:

- repeating work that was already done
- re-reading the same files every session
- losing the reason behind a change
- lacking a clean handoff between sessions
- relying on large prompts full of duplicated context
- wasting tokens on context reconstruction

Git alone does not solve this. Commit messages are often incomplete, and diff history is noisy. Plain notes are also not enough because they are not structured or repo-aware.

---

## Goal

Build a small, reliable CLI that agents can use to:

1. capture useful working context quickly
2. recall relevant context with minimal tokens
3. generate structured handoffs between sessions
4. search commit history semantically and by keyword
5. provide a predictable interface for both humans and LLMs

---

## Non-goals for MVP

The MVP will **not** include:

- a full TUI
- autonomous task execution
- multi-repo cloud sync
- IDE-specific UI
- team collaboration features
- background daemons
- automatic code modification
- a general second-brain product

These can be explored later if the core repo-memory workflow proves useful.

---

## Core product thesis

The main value of `aid` is not “notes in the terminal”.

The value is:

**making agents less forgetful, less repetitive, and more token-efficient inside a real codebase**

That means the product must optimise for:

- low-friction capture
- high-signal recall
- structured output
- machine-readable output
- predictable command behaviour

---

## Primary users

### Human developer
A developer working in a repo who wants quick notes, task state, decision history, and better Git recall.

### AI coding agent
An agent running in a repo that needs a standard way to:

- resume prior work
- store observations
- retrieve decisions
- understand recent changes
- generate a handoff

### Hybrid workflow
A human and one or more agents sharing the same local memory layer in a repo.

---

## MVP scope

The MVP will include five core object types:

- notes
- tasks
- decisions
- handoffs
- commit index

### Notes
Short observations, findings, reminders, and context fragments.

Notes are always repo-scoped and may optionally be tied to a branch.

Examples:

- “VAT mismatch appears during invoice line aggregation”
- “Klyant ledger endpoint is source of truth for balances”
- “Auth refresh logic is brittle in current implementation”

### Tasks
Explicit units of work with basic lifecycle state.

Examples:

- open
- in_progress
- done
- blocked

### Decisions
Structured records of important engineering choices.

Examples:

- “Store money as integer pence to avoid floating-point issues”
- “Use snapshotting for aggregate recovery speed”
- “Treat Klyant as external source of truth”

### Handoffs
Explicitly generated, saved summaries of current repo state for the next session, human or agent.

### Commit index
A searchable index over Git history with both keyword and semantic recall.

---

## Key workflows

### 1. Capture context

Fast write path with minimal ceremony.

```bash
aid note add "VAT bug likely caused by invoice line rounding"
aid task add "Fix VAT rounding on invoice lines"
aid decide add "Store all monetary values as integer pence"
```

### 2. Resume work

Agent or developer enters a repo and asks for the smallest useful context bundle.

```bash
aid resume
```

Expected result:

- current branch
- inferred active task when unambiguous
- latest notes
- recent decisions
- relevant recent commits
- latest saved handoff when useful
- suggested next step

### 3. Search history

```bash
aid history search "Klyant auth refresh"
aid recall "Why do we store money as integer pence?"
```

### 4. Generate handoff

```bash
aid handoff generate
```

Expected result:

- saved snapshot of current repo state
- branch name
- working tree state
- recent notes
- open tasks
- key decisions
- relevant commits
- open questions
- recommended next action

---

## Design principles

1. **Repo-first**  
   All memory is scoped to a Git repository by default.
2. **Agent-friendly**  
   Commands and outputs must be easy for an LLM to call and parse.
3. **Token-efficient**  
   Outputs should be concise by default and expandable when needed.
4. **Structured where it matters**  
   Human-readable by default, machine-readable when requested.
5. **Local-first**  
   The MVP should work entirely on a local machine with no cloud dependency.
6. **Boring technology**  
   Use simple, dependable components over clever infrastructure.

---

## Command design

The CLI should have a predictable top-level structure.

```text
aid <resource> <action> [options]
aid <shortcut-command> [options]
```

### Core commands for MVP

```text
aid init
aid status
aid resume
aid recall <query>

aid note add <text>
aid note list

aid task add <text>
aid task list
aid task done <id>

aid decide add <text>
aid decide list

aid handoff generate
aid handoff list

aid history index
aid history search <query>
```

### Optional aliases

Useful, but not required for first release.

```text
aid notes add ...
aid tasks add ...
aid decisions add ...
```

The product should prefer clarity over cleverness.

---

## Help interface requirements

The `--help` experience must be unusually good.

This matters because:

- humans will learn the tool through help text
- agents will also read the help text
- poor help output wastes tokens and causes misuse

### Requirements

- every command must support `--help`
- help output must be concise and structured
- examples must be short and realistic
- avoid decorative prose
- keep line lengths sensible for terminal display
- include common flags consistently

### Example top-level help

```text
aid - local memory for coding agents and repos

Usage:
  aid <command> [options]

Core commands:
  init                Initialise aid in the current repository
  status              Show repo memory status
  resume              Show a compact working summary
  recall <query>      Search notes, decisions, handoffs, and commits

  note add <text>     Add a note
  note list           List recent notes

  task add <text>     Add a task
  task list           List tasks
  task done <id>      Mark a task as done

  decide add <text>   Record an engineering decision
  decide list         List decisions

  handoff generate    Create a structured handoff summary
  handoff list        List saved handoffs

  history index       Index git history for search
  history search <q>  Search indexed commit history

Global options:
  --json              Output machine-readable JSON
  --brief             Use compact output
  --repo <path>       Operate on a specific repository
  --help              Show help for a command

Examples:
  aid resume
  aid note add "Refresh token bug occurs after 401 retry"
  aid recall "Why do we store money as integer pence?"
  aid history search "invoice VAT reconciliation"
```

### Help style rules

- use sentence fragments, not long paragraphs
- prefer one-line command descriptions
- keep examples directly related to real usage
- avoid exposing internal implementation details

---

## Output design

The CLI must support two output modes:

1. **Human-readable output**  
   Default terminal output for developers.
2. **Machine-readable output**  
   `--json` for agents, scripts, and editor integrations.

### Global output flags

- `--json`
- `--brief`
- `--verbose`

### Output rules

#### Default mode

- concise summary
- readable headings
- no unnecessary filler
- prioritise active and recent items

#### `--brief`

- smallest useful answer
- suitable for low-token agent workflows
- no explanatory text unless essential

#### `--json`

- stable schema
- no surrounding commentary
- deterministic field naming
- include IDs where useful
- use a versioned success/error envelope

### Example: `aid resume --brief`

```text
Branch: feat/vat-fix
Task: Fix VAT rounding on invoice lines
Notes:
- VAT mismatch reproduced in invoice aggregation
- Klyant values appear correct upstream
Decisions:
- Store money as integer pence
Recent commits:
- 8d12c3a fix: normalise invoice VAT calculation
Next:
- inspect invoice line subtotal rounding path
```

### Example: `aid resume --json`

```json
{
  "schema_version": "1",
  "ok": true,
  "command": "resume",
  "data": {
    "repo": {
      "name": "conveyancing-app",
      "path": "/path/to/conveyancing-app",
      "branch": "feat/vat-fix"
    },
    "active_task": {
      "id": "task_12",
      "text": "Fix VAT rounding on invoice lines",
      "status": "in_progress",
      "branch": "feat/vat-fix",
      "created_at": "2026-04-14T10:00:00Z",
      "updated_at": "2026-04-14T10:15:00Z"
    },
    "active_task_inferred": true,
    "notes": [
      {
        "id": "note_21",
        "text": "VAT mismatch reproduced in invoice aggregation",
        "branch": "feat/vat-fix",
        "scope": "branch",
        "created_at": "2026-04-14T09:40:00Z"
      },
      {
        "id": "note_22",
        "text": "Klyant values appear correct upstream",
        "branch": null,
        "scope": "repo",
        "created_at": "2026-04-14T09:43:00Z"
      }
    ],
    "decisions": [
      {
        "id": "decision_3",
        "text": "Store money as integer pence",
        "rationale": null,
        "branch": "feat/vat-fix",
        "created_at": "2026-04-13T16:20:00Z"
      }
    ],
    "recent_commits": [
      {
        "sha": "8d12c3a",
        "summary": "fix: normalise invoice VAT calculation",
        "message": "fix: normalise invoice VAT calculation",
        "author": "Dan",
        "committed_at": "2026-04-14T08:55:00Z",
        "changed_paths": [
          "app/invoices.py"
        ]
      }
    ],
    "latest_handoff": null,
    "next_action": "inspect invoice line subtotal rounding path"
  },
  "error": null
}
```

---

## Token efficiency requirements

Token efficiency is a first-class product requirement.

### Goals

- minimise repeated context
- compress output without losing meaning
- avoid boilerplate
- allow agents to ask for more detail only when needed

### Practical rules

#### Input efficiency

- commands should be short and consistent
- avoid verbose required arguments
- sensible defaults should reduce prompt size

#### Output efficiency

- brief mode should be highly compressed
- fields should be prioritised by relevance
- avoid repeating branch, repo, or timestamps unnecessarily
- do not include empty sections
- truncate long content by default

#### Retrieval efficiency

- search results should return top matches only by default
- allow `--limit`
- rank current branch and recent items higher
- prioritise summaries over raw diffs

#### Agent efficiency

- `--json` output should be stable and compact
- provide only the fields an agent needs
- keep schema flat where possible
- use `null` for missing single values and `[]` for empty collections
- use RFC 3339 UTC timestamps

---

## Storage and architecture

### Local storage

Use SQLite for the MVP.

The primary database should live in the user's local app-data directory, not inside the repository.

Suggested defaults:

- macOS: `~/Library/Application Support/aid/aid.db`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/aid/aid.db`
- Windows: `%LocalAppData%\\aid\\aid.db`

### Why SQLite

- zero external setup
- portable
- fast enough
- easy to inspect and debug
- supports structured tables and FTS

### Suggested data model

#### `repos`

- `id`
- `path`
- `name`
- `created_at`
- `updated_at`

#### `notes`

- `id`
- `repo_id`
- `branch`
- `scope`
- `text`
- `tags`
- `created_at`

#### `tasks`

- `id`
- `repo_id`
- `branch`
- `scope`
- `text`
- `status`
- `created_at`
- `updated_at`

#### `decisions`

- `id`
- `repo_id`
- `branch`
- `text`
- `rationale`
- `created_at`

#### `handoffs`

- `id`
- `repo_id`
- `branch`
- `summary`
- `created_at`

#### `commits`

- `id`
- `repo_id`
- `sha`
- `author`
- `committed_at`
- `message`
- `changed_paths`
- `summary`
- `indexed_at`

#### `search_chunks`

- `id`
- `repo_id`
- `source_type`
- `source_id`
- `text`
- `embedding`
- `created_at`

---

## Search strategy

Use a hybrid search model.

### Phase 1

- SQLite FTS for keyword search
- relevance boosts for current repo, current branch, and recent records

### Phase 2

- optional embeddings for semantic search
- embeddings applied to notes, decisions, handoffs, and commit summaries

### Why this approach

FTS gives a strong baseline with low complexity. Embeddings can be added later without making the MVP dependent on remote infrastructure.

---

## Git integration

The MVP should integrate with Git but avoid indexing everything in the most expensive way possible.

```bash
aid history index
```

This command should gather and store:

- commit SHA
- author
- date
- commit message
- changed file paths
- short heuristic summary

### Summary strategy

Do not embed raw diffs in full by default.

Instead:

1. capture commit metadata
2. prefer the commit subject when it is already useful
3. otherwise derive a deterministic summary from metadata and changed paths
4. store changed paths
5. optionally embed the summary

This keeps the index smaller and more useful.

### Search examples

```bash
aid history search "token refresh retry"
aid history search "invoice VAT"
aid history search "why was soft delete added to payment intents"
```

---

## Agent skill file

A core part of the product is shipping a static Agent Skills-compatible skill package with the repo so the user can give their agent clear instructions on how to use `aid`.

This file should explain:

- what `aid` is for
- when to call it
- which commands to prefer
- how to keep output token-efficient
- how to write notes and decisions well

The skill should be version-controlled and maintained like any other repo artifact. It should not be generated dynamically by `aid`.

Users can install it into compatible agents with `npx skills add` or `bunx skills add`, or by copying it manually.

### Proposed format

A standard skill package centered on:

- `SKILL.md`

Suggested repo path:

- `.agents/skills/aid/SKILL.md`

### Skill file objectives

- teach the agent to call `aid resume` at session start
- teach the agent to write notes only when useful
- teach the agent to use `--brief` or `--json` when appropriate
- teach the agent to record decisions explicitly
- teach the agent to generate a handoff at session end

### Example skill file outline

```md
# Using aid in this repository

## Purpose
Use `aid` to manage working context in this repository.

## Start of session
Run:
- `aid resume --brief`

If more structure is needed, run:
- `aid resume --json`

## During work
Record only important findings:
- `aid note add "<finding>"`

Track meaningful tasks:
- `aid task add "<task>"`

Record non-obvious engineering decisions:
- `aid decide add "<decision>"`

## Searching context
Use:
- `aid recall "<query>"`
- `aid history search "<query>"`

## End of session
Generate a handoff:
- `aid handoff generate --brief`

## Efficiency rules
- Prefer `--brief` unless detailed output is required
- Prefer targeted queries over broad recall
- Do not spam notes for trivial observations
- Record decisions when they affect future implementation
```

---

## Repository initialisation

`aid init` should:

1. verify the current directory is inside a Git repository
2. create local storage if needed
3. attach the repo to the local `aid` database
4. optionally create a repo-local config file

### Possible generated files

```text
.aid/config.toml
```

The repo-local config should remain simple in the MVP.

---

## Configuration

The MVP should support minimal configuration.

### Global config

User-level defaults such as:

- output mode preferences
- embedding provider settings
- indexing defaults

### Repo config

Repo-specific settings such as:

- preferred summary depth
- ignored paths for indexing
- agent instructions

### Example repo config

```toml
[output]
default_mode = "brief"

[indexing]
ignore_paths = ["vendor/", "node_modules/", "storage/"]

[agent]
skill_path = ".agents/skills/aid/SKILL.md"
```

---

## MVP acceptance criteria

The MVP is complete when a user can:

1. initialise `aid` in a Git repo
2. add and list notes
3. add, list, and complete tasks
4. add and list decisions
5. generate a useful handoff
6. index Git history
7. search indexed history
8. run `aid resume` and get a compact, useful context summary
9. use `--json` output for scripting or agents
10. include a static `SKILL.md` skill package in the repo

---

## Success criteria

The MVP is successful if it reduces friction in real agent workflows.

### Signs of success

- developers actually call `aid resume` at the start of work
- agents use `aid` without needing long manual prompting
- handoffs are useful enough to be reused
- decision recall prevents repeated analysis
- history search surfaces relevant commits faster than plain `git log`
- brief output is good enough to save tokens in repeated use

### Signs of failure

- users treat it as a generic note dump
- outputs are too verbose for agents
- help text is too vague to guide correct usage
- history indexing is expensive but low-value
- no one uses handoff generation
- `aid resume` does not reliably save time

---

## Suggested implementation order

### Phase 1

- `aid init`
- `aid note add`
- `aid note list`
- `aid task add`
- `aid task list`
- `aid task done`
- `aid decide add`
- `aid decide list`

### Phase 2

- `aid resume`
- `aid handoff generate`
- top-level `--help`
- per-command `--help`

### Phase 3

- `aid history index`
- `aid history search`
- SQLite FTS

### Phase 4

- `--json`
- `--brief`
- static `SKILL.md` skill package

### Phase 5

- optional embeddings
- relevance tuning
- better ranking and summarisation

---

## Resolved MVP decisions

1. Notes are always stored against a repository and may optionally be tied to a branch. `aid note add` should default to the current branch, with repo-wide notes being explicit.
2. `aid resume` should infer the active task when there is a single clear candidate. If task state is ambiguous, it should say so instead of guessing.
3. Handoffs are explicit saved snapshots. `aid handoff generate` both displays and stores the generated handoff.
4. Commit summaries are heuristic in the MVP. Use the existing commit subject when it is good enough and fall back to deterministic path- and metadata-based summarisation.
5. The primary SQLite database lives in the user's local app-data directory by default. Repositories may contain lightweight config, but not the main database.
6. Agent instructions are shipped as a static Agent Skills-compatible `SKILL.md` package in the repo. `aid` does not generate this file dynamically.
7. `--json` output uses a versioned, stable envelope with consistent resource objects, explicit nulls, and command-specific payloads.

---

## Recommended MVP positioning

`aid` should be positioned as:

> a local memory CLI for coding agents and developers working inside Git repos

Not:

- an AI shell
- a task execution engine
- an IDE replacement
- a knowledge base for everything

The sharp positioning matters because it keeps the product coherent.

---

## Summary

The MVP for `aid` is a narrow, useful CLI that helps agents and developers preserve and recover repo context with minimal friction.

Its defining features are:

- compact memory capture
- strong resume workflow
- structured handoffs
- searchable commit history
- excellent help output
- agent-oriented skill file
- token-efficient input and output

If the MVP works, `aid` becomes the first command an agent runs when entering a repository, and the last command it runs before leaving.

The next sensible step is to turn this into a polished `README.md` plus a first draft of a static `SKILL.md` package.
