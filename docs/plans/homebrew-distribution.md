# Homebrew Distribution Plan

Status: proposed
Last updated: 2026-04-14

## Goal

Add a first-class Homebrew install path for `aid` without making the existing GitHub release flow harder to operate.

The desired end state is:

1. macOS users can install `aid` with a single Homebrew command.
2. Formula updates track new `aid` releases automatically.
3. The existing release pipeline and `install.sh` remain the primary direct-binary path.
4. Homebrew support does not require manual tap edits on every release.

## Current State

- `release-please` and GoReleaser already create GitHub Releases and publish archives plus `checksums.txt`.
- `scripts/install.sh` already installs the matching prebuilt binary for macOS and Linux.
- There is no Homebrew tap repository yet.
- There is no signed/notarized macOS binary distribution story yet.

## Recommendation

Use a dedicated tap repository, `forjd/homebrew-tap`, with a standard Homebrew formula that builds `aid` from source tags.

Automate formula updates from this repository after each published release.

This is the best fit for the repo right now:

- it keeps Homebrew as a convenience install surface, not a second release system
- it avoids depending on GoReleaser's deprecated `brews` path
- it avoids macOS quarantine and signing concerns that come with binary casks
- it works for both Homebrew on macOS and Homebrew on Linux
- it leaves `scripts/install.sh` as the prebuilt-binary path for users who want the fastest install

## Why Not the Alternatives

### GoReleaser-managed `brews`

This used to be the obvious option, but it is now deprecated upstream in favor of casks.

Using it here would add automation on top of a path that is already being phased out. That is unnecessary maintenance debt for a small CLI.

### Homebrew cask right now

A cask would install the prebuilt macOS binary, which sounds attractive, but it is not the right first move for this repo.

Main drawbacks:

- casks are macOS-focused and do not help Linux Homebrew users
- unsigned and non-notarized binaries can trigger quarantine friction
- working around quarantine weakens the trust story and creates another operational concern

If `aid` later gets signed and notarized macOS binaries, revisiting a cask can make sense.

### `homebrew/core`

Publishing to `homebrew/core` is a later-stage optimization, not the first step.

Main drawbacks:

- slower update path
- more review and policy overhead
- less control over rollout while the CLI is still changing quickly

## Proposed Install Model

### User-facing install

Recommended command:

```bash
brew install forjd/tap/aid
```

This keeps the install UX to one command while still using an upstream-owned tap.

### Distribution split

Use each install surface for what it is best at:

- Homebrew: one-command install and upgrade for users already on Homebrew
- `scripts/install.sh`: direct install of prebuilt release binaries
- manual GitHub Release downloads: explicit fallback for users who do not want either tool

## Proposed Tap Repository

Create a new public repository:

- `forjd/homebrew-tap`

Recommended initial contents:

- `Formula/aid.rb`
- `README.md`

Repository conventions:

- default branch: `main`
- formula directory: `Formula/`
- install command: `brew install forjd/tap/aid`

## Recommended Formula Shape

The formula should build from a tagged source archive, not download a prebuilt binary.

Use this exact source archive URL shape:

- `https://github.com/forjd/aid/archive/refs/tags/vX.Y.Z.tar.gz`

Expected formula behaviour:

- `desc`, `homepage`, and `license` are set explicitly
- source URL points at the tagged `aid` source archive for `vX.Y.Z`
- `sha256` is pinned to that source archive
- `depends_on "go" => :build`
- build uses `go build -trimpath -o bin/aid ./cmd/aid`
- installed binary name is `aid`
- test block executes a stable command that Homebrew can assert on, such as `aid --help`

This keeps the formula conventional, auditable, and independent from archive naming details in the release assets.

## Proposed Automation

### Trigger

Add a workflow in the main repository that runs when a GitHub Release is published.

Recommended trigger:

- `release`
- types: `[published]`

Using the published release event keeps the tap update tied to the same release boundary users already recognize.

### Workflow responsibilities

The workflow should:

1. read the release tag, such as `v0.2.0`
2. compute the source archive URL for that tag as `https://github.com/forjd/aid/archive/refs/tags/<tag>.tar.gz`
3. download the source archive and calculate its SHA-256
4. render `Formula/aid.rb` from a small template or checked-in script
5. validate the rendered formula with `brew audit --strict`, `brew install --build-from-source`, and `brew test aid`
6. commit and push the updated formula to `forjd/homebrew-tap`

### Push strategy

The workflow should push directly to `main` in `forjd/homebrew-tap` after validation passes.

This keeps the release-to-tap path automatic and avoids leaving a published release waiting on a second manual merge step.

### Failure behaviour

If formula rendering or validation fails, the tap update job should fail loudly and leave the existing tap unchanged.

This should not delete or rewrite the published GitHub Release. The operational expectation is:

- the release remains available through GitHub Releases and `install.sh`
- Homebrew stays on the previous known-good formula until the failure is fixed
- the workflow failure is the signal to follow up manually

### Auth

This workflow should use a dedicated token, not the default `GITHUB_TOKEN`.

Recommended secret:

- `HOMEBREW_TAP_TOKEN`

That token needs write access to `forjd/homebrew-tap`.

## Proposed Files

In this repository:

- `docs/plans/homebrew-distribution.md`
- `.github/workflows/homebrew-tap.yml`
- either:
  - `scripts/render_homebrew_formula.sh`
  - or `.github/homebrew/aid.rb.tmpl`
- `README.md`
- `CONTRIBUTING.md`

In `forjd/homebrew-tap`:

- `Formula/aid.rb`
- `README.md`

## Release-to-Tap Flow

The full intended flow is:

1. feature or fix work merges to `main`
2. `release-please` opens or updates the release PR
3. merging the release PR creates the tag and GitHub Release
4. the Homebrew workflow runs on `release.published`
5. the workflow validates and updates `forjd/homebrew-tap/Formula/aid.rb`
6. users can install or upgrade with Homebrew

This keeps the Homebrew update downstream from the release boundary instead of mixing tap logic into the main release job.

## Implementation Steps

### Phase 1: create the tap repo

Create `forjd/homebrew-tap` with:

- `Formula/`
- `README.md`

Success condition:

- `brew install forjd/tap/aid` is the intended final command shape

### Phase 2: author and validate the initial formula

Create the first `Formula/aid.rb` using a released tag.

Validate locally on a Homebrew machine with:

- `brew audit --strict ./Formula/aid.rb`
- `brew install --build-from-source ./Formula/aid.rb`
- `brew test aid`
- `aid --help`

Success condition:

- the formula passes audit, installs cleanly from source, and produces a working `aid` binary

### Phase 3: automate formula updates

Add a workflow in the main repo to:

- trigger on `release.published`
- compute the source archive checksum
- update the formula in `forjd/homebrew-tap`
- validate the rendered formula before publishing it
- push the change with `HOMEBREW_TAP_TOKEN`

Success condition:

- a newly published release updates the tap without manual edits when validation passes, and leaves the previous formula untouched when validation fails

### Phase 4: document the new install path

Update docs so install options are clearly ordered:

- Homebrew for Homebrew users
- `install.sh` for direct prebuilt-binary installs
- manual GitHub Release download as the explicit fallback

Success condition:

- README install docs are concise and do not present overlapping paths as equally primary

### Phase 5: rollout validation

Validate in order:

1. publish a new release
2. confirm the tap repo updates to the new version
3. on a clean macOS machine, run `brew install forjd/tap/aid`
4. verify `aid --help`
5. publish a second release and confirm `brew upgrade` picks it up

Recommended extra validation:

- verify the formula also builds on Linux Homebrew

## Operational Notes

### Keep Homebrew conventional

The Homebrew formula should stay source-based unless there is a strong reason to move away from that model.

That keeps the tap simple and avoids coupling the formula to signed binary distribution work.

### Keep the tap small

Do not turn the tap into a second product surface.

At this stage it only needs:

- one formula
- one README
- straightforward automation

### Revisit casks later if distribution changes

If the repo later ships signed and notarized macOS binaries, and Homebrew becomes the dominant macOS install path, add a separate follow-up plan to evaluate a cask.

That should be a distinct decision, not bundled into the first Homebrew rollout.
