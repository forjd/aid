---
name: aid
description: Capture and recall working context with the `aid` CLI in this repository. Use when resuming prior work, recording important findings, tracking meaningful tasks, storing engineering decisions, searching saved context, or generating a handoff for the next session.
compatibility: Intended for this repository and requires the local `aid` CLI to be available.
---

# Using aid in this repository

## Purpose

Use `aid` to capture and recall useful working context for this repository.

The tool is intended to reduce repeated context reconstruction across human and agent sessions.

## Current state

This repository currently contains a scaffold.

Prefer the CLI help output and repository docs if implementation details differ from the intended command shape below.

## Start of session

Run:

- `aid resume --brief`

If more structure is required, use:

- `aid resume --json`

## During work

Record only important findings:

- `aid note add "<finding>"`

Track meaningful units of work:

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

## Writing rules

- Prefer short, specific notes over vague summaries.
- Record decisions when they change future implementation.
- Do not spam notes for trivial observations.
- Prefer targeted recall queries over broad searches.
- Prefer `--brief` unless richer structure is needed.
