# Automated SemVer Release Plan

Status: proposed
Last updated: 2026-04-14

## Goal

Add GitHub-based release automation so that commits merged to `main` drive versioned SemVer releases, with GitHub Releases publishing prebuilt `aid` binaries for the main operating systems.

The desired end state is:

1. Regular feature and fix work merges into `main`.
2. GitHub Actions opens or updates a release PR automatically.
3. Merging that release PR creates a SemVer tag and GitHub Release.
4. The release publishes multi-platform Go binaries and checksums automatically.

## Current State

- The repo has no `.github/workflows/`.
- The GitHub repo currently has no Actions workflows, tags, or releases.
- The repo already uses Go modules and the canonical module path `github.com/forjd/aid`.
- Commit history is partly Conventional Commit-shaped (`feat: ...`, `chore: ...`) but not fully consistent yet.

## Recommendation

Use:

- `release-please` to decide the next SemVer version, maintain `CHANGELOG.md`, open/update the release PR, and create the GitHub release/tag.
- `GoReleaser` to build and upload the release artifacts for multiple OS/architecture targets.

This is the cleanest fit for a Go CLI on GitHub:

- `release-please` is GitHub-native and works well for Conventional Commit-driven release automation.
- `GoReleaser` is the standard Go release tool for cross-platform binaries, archives, and checksums.
- The split keeps versioning/changelog concerns separate from packaging/publishing concerns.

## Why Not the Alternatives

### GoReleaser only

This is fine when maintainers are happy to create tags manually. It does not satisfy the requirement to create versioned releases automatically from commits landing on `main`.

### `semantic-release`

It can work, but it is more JavaScript-centric and adds Node tooling without giving this repo much over `release-please` for a small Go CLI.

## Proposed Release Flow

### Workflow A: release planning on `main`

Trigger:

- `push` to `main`

Responsibilities:

- inspect commit history since the last release
- open or update a release PR when releasable changes exist
- when the merged commit is itself the release PR merge, create:
  - a SemVer tag such as `v0.1.0`
  - a GitHub Release
  - an updated `CHANGELOG.md`

Recommended implementation:

- `.github/workflows/release-please.yml`
- `googleapis/release-please-action`
- manifest mode, root component only
- `release-type: go`
- `include-v-in-tag: true`

### Workflow B: artifact publishing on tags

Trigger:

- `push` tags matching `v*.*.*`

Responsibilities:

- check out the tagged source
- build release artifacts for the selected OS/architecture matrix
- generate checksums
- upload archives and checksum files to the GitHub Release created by Workflow A

Recommended implementation:

- `.github/workflows/release-artifacts.yml`
- `goreleaser/goreleaser-action`
- `.goreleaser.yaml`

## Important Token/Trigger Constraint

This is the main implementation detail that matters.

If `release-please` creates the tag and release with the default repository `GITHUB_TOKEN`, the follow-on tag workflow will not fire. GitHub suppresses new workflow runs from events created by that token.

Because this repo wants:

- automated release creation on `main`, and
- automated binary publishing when that release is created,

the plan should use a dedicated token for `release-please`, not the default `GITHUB_TOKEN`.

Recommended options:

1. Preferred practical option: a fine-grained PAT stored as `RELEASE_PLEASE_TOKEN`.
2. Better long-term org option: a GitHub App token with equivalent repository permissions.

That token should be used only in the `release-please` workflow. The tag-triggered artifact workflow can use the normal `GITHUB_TOKEN`.

## Proposed Files

- `.github/workflows/release-please.yml`
- `.github/workflows/release-artifacts.yml`
- `release-please-config.json`
- `.release-please-manifest.json`
- `.goreleaser.yaml`
- `CHANGELOG.md`

Docs to update during implementation:

- `README.md`
- `CONTRIBUTING.md`

## Proposed Config Shape

### `release-please`

Use manifest mode even though this repo has one component. It keeps future changes predictable and avoids having to rework config later.

Suggested defaults:

- root path: `.`
- release type: `go`
- tag format: `vX.Y.Z`
- changelog path: `CHANGELOG.md`
- package name: `aid`

Initial version choice:

- default recommendation: bootstrap at `v0.1.0`
- fallback if maintainers want a smaller public signal: `v0.0.1`

`v0.1.0` is the better default here because the repo already has a usable CLI surface and documented MVP workflow.

### `GoReleaser`

Start with a conservative binary matrix:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

Optional later:

- `windows/arm64`
- Linux package formats
- Homebrew/Scoop publishing
- signing, SBOMs, or provenance attestation

Artifact expectations:

- `.tar.gz` archives for Unix targets
- `.zip` archives for Windows
- `checksums.txt`

Recommended GoReleaser behavior:

- keep existing release notes instead of replacing them
- upload artifacts to the GitHub Release created by `release-please`
- avoid introducing extra packaging surfaces in the first pass

## Commit Message Policy

This plan assumes Conventional Commit semantics for future merges to `main`.

Minimum rules:

- `feat:` bumps minor
- `fix:` bumps patch
- `!` or `BREAKING CHANGE:` drives major
- `docs:`, `chore:`, `test:`, and `ci:` should not normally trigger a release

This repo should prefer squash merges with a reviewed PR title or merge commit title so the release signal stays clean.

Implementation should update `CONTRIBUTING.md` with 4-5 concrete examples.

## Implementation Steps

### Phase 1: repo scaffolding

Add:

- `docs/plans/automated-semver-releases.md` as the approved plan
- `CHANGELOG.md`
- `release-please-config.json`
- `.release-please-manifest.json`
- `.goreleaser.yaml`
- `.github/workflows/release-please.yml`
- `.github/workflows/release-artifacts.yml`

### Phase 2: release config

Configure `release-please` to:

- watch `main`
- manage the root component
- write `CHANGELOG.md`
- create `v`-prefixed tags
- use `RELEASE_PLEASE_TOKEN`

### Phase 3: binary config

Configure GoReleaser to:

- build the `aid` binary from `./cmd/aid`
- cross-compile for the chosen target matrix
- emit archives and checksums
- publish assets to the existing GitHub Release

### Phase 4: docs and contributor guidance

Update docs so contributors know:

- release tags are automated
- manual tagging is no longer the default
- merge titles should follow Conventional Commit format

### Phase 5: rollout validation

Validate in order:

1. merge a non-release commit path and confirm no release PR is created
2. merge a `fix:` or `feat:` change and confirm a release PR appears
3. merge the release PR and confirm:
   - a SemVer tag is created
   - a GitHub Release appears
   - GoReleaser uploads binaries and checksums
4. download at least one Linux, macOS, and Windows artifact and verify naming/layout

## Operational Notes

### Branch protection

If branch protection is enabled later, keep `main` protected and let the release PR merge through the normal review path.

### Tests

The artifact workflow may run `go test ./...` before publishing, but the cleaner setup is:

- a separate CI workflow owns test feedback on PRs and `main`
- the release artifact workflow focuses on reproducible builds and publishing

That CI workflow is adjacent work, not required to ship the release flow.

### Version embedding

This plan does not require a runtime `aid version` command.

If the CLI later adds one, GoReleaser should inject:

- version
- commit SHA
- build date

via `-ldflags`.

## Risks

### Inconsistent merge titles

If maintainers merge PRs with vague titles like `update stuff`, release automation will produce weak or missing release signals.

Mitigation:

- document Conventional Commit expectations
- prefer squash merge with curated PR titles

### Token setup errors

If `release-please` uses the default `GITHUB_TOKEN`, the tag-triggered artifact workflow will not run.

Mitigation:

- require `RELEASE_PLEASE_TOKEN` before enabling the workflows
- test the first release in a controlled rollout

### Cross-platform build surprises

The repo uses `modernc.org/sqlite`, so the first release should explicitly verify the chosen target matrix rather than assuming every target behaves identically.

Mitigation:

- keep the initial matrix modest
- validate Linux, macOS, and Windows downloads after the first release

## Acceptance Criteria

- A push to `main` can open or update a release PR automatically.
- Merging the release PR creates a SemVer tag with a `v` prefix.
- A GitHub Release is created automatically with the release notes from `release-please`.
- Release artifacts are uploaded automatically for the chosen platforms.
- A checksum file is published with the release assets.
- No manual tag creation is required for normal releases.

## Follow-up Work That Is Explicitly Out of Scope

- Homebrew formula publishing
- Scoop manifest publishing
- package repositories (`deb`, `rpm`, `apk`)
- signing, provenance, or notarization
- Docker images
- prerelease channels (`rc`, `beta`, nightly)

Those can be layered on later after the basic SemVer release loop is stable.
