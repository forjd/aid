# Code Coverage Improvement Plan

Status: proposed
Last updated: 2026-04-14

## Goal

Raise confidence in the repo by improving direct, behavior-focused test coverage across the packages that currently rely on incidental coverage from the CLI suite.

The desired end state is:

1. Coverage is measured the same way locally and in CI.
2. Core packages have their own focused tests instead of depending mostly on end-to-end CLI flows.
3. Zero-coverage and low-coverage branches around parsing, rendering, ranking, and recovery paths are explicitly exercised.
4. Coverage on touched code stops drifting down over time.

## Baseline

Baseline captured on 2026-04-14 with:

- `go test ./...`
- `go test ./... -coverpkg=./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Current snapshot:

- whole-repo statement coverage: `72.3%`
- packages in repo: `12`
- packages with tests: `2`
- Go source files: `26`
- test files: `2`
- package-local coverage:
  - `internal/cli`: `70.1%`
  - `internal/store/sqlite`: `71.2%`

This number is better than the file count suggests because the CLI tests exercise several packages transitively. The weakness is that much of that coverage is indirect. A refactor inside `output`, `resume`, `handoff`, `search`, `config`, or `git` could regress behavior without a package-local test making the failure obvious.

## What the Baseline Is Telling Us

The suite already covers the main happy paths:

- repo init and CRUD flows through the CLI
- JSON, brief, and verbose output for key commands
- SQLite CRUD and FTS-backed search behavior

The biggest gaps are not the basic flows. They are:

- direct unit tests for pure or mostly pure packages
- branch coverage for error paths and fallback logic
- explicit tests for ranking and filtering heuristics
- recovery paths in SQLite FTS maintenance

Known `0%` functions as of the baseline:

- `cmd/aid/main.go: main`
- `internal/cli/cli.go: lookupHelpTarget`
- `internal/cli/cli.go: inferCommandPath`
- `internal/cli/cli.go: stubCommand`
- `internal/cli/handlers.go: filterIndexedCommits`
- `internal/output/render.go: WriteError`
- `internal/output/render.go: RenderTaskCompleted`
- `internal/output/render.go: commandName`
- `internal/resume/resume.go: branchRank`
- `internal/store/sqlite/store.go: rebuildDecisionFTS`
- `internal/store/sqlite/store.go: rebuildHandoffFTS`

## Coverage Targets

Use targets to improve the suite without chasing meaningless numbers.

Phase targets:

- short term: raise whole-repo coverage from `72.3%` to at least `80%`
- medium term: raise whole-repo coverage to at least `85%`
- package-local target for `internal/cli` and `internal/store/sqlite`: at least `80%`
- add direct tests for `internal/output`, `internal/resume`, `internal/handoff`, `internal/search`, `internal/config`, and `internal/git`

Non-goals:

- forcing `100%` coverage
- adding tests that only mirror implementation details
- spending time on trivial wrappers unless they protect important behavior

## Recommended Strategy

Prioritize packages in this order:

### 1. `internal/output`

This package has a large amount of branchy, deterministic rendering logic and should be cheap to test directly. It is also the easiest place to turn incidental CLI assertions into stable package-level assertions.

Focus on:

- `WriteError`
- `RenderTaskCompleted`
- `RenderTasks`
- `RenderNoteAdded`
- `RenderDecisionAdded`
- `commandName`
- JSON rendering shape and empty-state handling

### 2. `internal/resume`, `internal/handoff`, and `internal/search`

These packages encode product behavior and ranking logic. They are important enough to deserve direct tests rather than only transitive coverage.

Focus on:

- active-task inference
- branch ranking and note/decision ranking
- carry-forward question behavior
- next-action inference
- handoff summary and open-question synthesis
- search result ordering and per-section limits

### 3. `internal/config` and `internal/git`

These packages are small but behaviorally important. They are good candidates for focused table-driven tests.

Focus on:

- config parsing with inline comments and list handling
- repo config defaults and invalid input handling
- git output parsing
- command failures and edge conditions in `runGitOutput`, `Root`, `Branch`, and `Status`

### 4. `internal/cli`

The CLI package already has decent happy-path coverage, so the next gains should target untested dispatch and help behavior rather than more end-to-end CRUD duplication.

Focus on:

- help target lookup
- inferred command paths
- stub command rendering
- history filtering around indexed commits
- invalid argument combinations and error exits

### 5. `internal/store/sqlite`

SQLite coverage is already respectable, but the remaining value is in negative and recovery paths.

Focus on:

- FTS rebuild paths for decisions and handoffs
- invalid task status handling
- sync behavior when commit sets shrink or partially overlap
- migration and schema-version edge cases
- malformed or missing rows where scan and parse behavior matters

### 6. `cmd/aid`

This is low priority. A smoke test through `go run ./cmd/aid --help` in CI is enough unless the entrypoint gains more behavior.

## Implementation Plan

### Phase 1: add coverage visibility

Add a standard local coverage workflow so contributors do not invent their own commands.

Recommended changes:

- add a `test-cover` Make target that runs:
  - `go test ./... -coverpkg=./... -coverprofile=coverage.out`
  - `go tool cover -func=coverage.out`
- document the coverage command in `README.md` or `CONTRIBUTING.md`
- optionally add a lightweight CI job that uploads or prints the coverage summary without failing the build yet

Success condition:

- any contributor can reproduce the same coverage number locally and in CI

### Phase 2: close the zero-coverage functions

Add narrowly scoped tests for the current `0%` functions before broadening the suite.

Recommended work:

- unit tests for `internal/output`
- targeted CLI tests for help path inference
- direct tests for `branchRank`
- SQLite tests that force decision and handoff FTS rebuilds

Success condition:

- all currently known `0%` functions are covered

### Phase 3: add direct package-level suites

Create package-local tests for behavior-heavy packages that are currently only exercised through the CLI.

Recommended new test files:

- `internal/output/render_test.go`
- `internal/resume/resume_test.go`
- `internal/handoff/handoff_test.go`
- `internal/search/search_test.go`
- `internal/config/config_test.go`
- `internal/git/*.go` tests as appropriate

Success condition:

- the repo reaches at least `80%` whole-repo coverage
- at least four currently indirect packages have direct tests

### Phase 4: strengthen error and fallback paths

Once the main package-level suites exist, fill in the recovery logic and low-frequency branches.

Recommended work:

- failing writer and malformed data cases in output
- ranking ties and empty-input cases in resume/search/handoff
- git command failure simulation
- SQLite migration and repair-path tests

Success condition:

- `internal/cli` and `internal/store/sqlite` each reach at least `80%` package-local coverage
- whole-repo coverage reaches at least `85%`

### Phase 5: add guardrails

After the suite is stable, prevent easy regression.

Recommended policy:

- CI must run the same `test-cover` command
- first gate: fail CI if whole-repo coverage drops below `80%`
- later gate: raise the floor to `85%`
- require no coverage decrease in packages touched by a PR when the change is mainly logic, parsing, ranking, or rendering

Success condition:

- coverage stops drifting downward between feature changes

## Test Design Rules

To keep the suite useful:

- prefer table-driven tests for parsing, ranking, and formatting branches
- assert user-visible behavior, not private implementation details
- keep end-to-end CLI tests for command wiring, not for every rendering branch
- use package-level tests for pure logic and deterministic renderers
- add regression tests when fixing a bug or coverage hole, instead of bulk snapshot churn

## Recommended Rollout Order

1. Add `test-cover` tooling and docs.
2. Write `internal/output` tests.
3. Write `internal/resume`, `internal/handoff`, and `internal/search` tests.
4. Add focused `internal/config` and `internal/git` tests.
5. Fill the remaining `internal/cli` and `internal/store/sqlite` gaps.
6. Add CI thresholds only after the suite is stable.

## Acceptance Checklist

- [ ] coverage is measured with a documented single command
- [ ] all current `0%` functions have tests
- [ ] direct tests exist for `output`, `resume`, `handoff`, `search`, `config`, and `git`
- [ ] whole-repo coverage is at least `80%`
- [ ] package-local coverage for `internal/cli` is at least `80%`
- [ ] package-local coverage for `internal/store/sqlite` is at least `80%`
- [ ] CI reports coverage and prevents obvious regression

## Recommendation

The best next move is not to add more broad CLI integration tests. It is to add direct tests around `internal/output`, `internal/resume`, and `internal/handoff` first. Those packages are rich in deterministic logic, they currently rely too much on incidental coverage, and they should move the coverage number materially with less brittleness than expanding the end-to-end suite.
