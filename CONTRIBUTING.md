# Contributing

Thanks for contributing to `aid`.

This project is intentionally small, local-first, and conservative about scope. Good contributions usually make the CLI clearer, the memory workflow more reliable, or the implementation simpler to maintain.

## Before You Start

- Read [README.md](README.md) for the project overview.
- Read [docs/mvp-status.md](docs/mvp-status.md) for current implementation status.
- Read [docs/spec/mvp.md](docs/spec/mvp.md) for product scope and non-goals.
- Check [docs/architecture.md](docs/architecture.md) before changing package boundaries.

## Development Setup

Requirements:

- Go 1.26+
- Git

Common commands:

```bash
make build
make fmt
make test
go run ./cmd/aid --help
```

## Contribution Guidelines

- Keep the product small and repo-focused.
- Prefer boring, maintainable solutions over clever abstractions.
- Do not expand the MVP into a framework, daemon, or hosted service.
- Preserve the CLI's predictable human-readable and JSON output surfaces.
- Keep rendering concerns out of storage and domain packages.
- Update docs when command behavior, workflow, or scope changes.

## Code Style

- Follow the existing package boundaries in `internal/`.
- Keep commands and flags explicit.
- Add tests for behavior changes when practical.
- Run `make fmt` and `make test` before opening a pull request.

## Pull Requests

When opening a pull request:

- Explain the problem being solved.
- Keep the change narrowly scoped.
- Note any user-facing CLI or output changes.
- Include documentation updates when relevant.
- Mention any follow-up work that is intentionally out of scope.

## Good First Contributions

- Tighten CLI help text.
- Improve recall or ranking quality without widening scope.
- Add tests around command behavior and output.
- Clean up docs, examples, and contributor ergonomics.

## Questions and Scope

If a change adds major new surface area, document the tradeoff first before implementing it. `aid` should stay sharp-edged and easy to reason about.
