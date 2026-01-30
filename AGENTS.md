# Repository Guidelines

## Project Structure & Module Organization
- `cmd/prompter/` holds the Cobra CLI entrypoint and subcommands (e.g., `main.go`, `root.go`, `generate.go`).
- `internal/` contains app wiring, workflows, domain models, adapters, and utilities.
- `scripts/` provides build/release helpers; `.github/workflows/` hosts release automation.
- `example-config.toml`, `README.md`, and `INSTALL.md` document usage and configuration.

## Build, Test, and Development Commands
- `make build` builds the current platform binary.
- `make dev-build` builds without optimizations for local debugging.
- `make test` runs all Go tests.
- `make cross-platform` builds multi-platform binaries via `scripts/build.sh`.
- `make install` installs to `/usr/local/bin`.

## Coding Style & Naming Conventions
- Use Go standard formatting (`gofmt`) and idiomatic Go style.
- Indentation: tabs in Go source, 2 spaces in Markdown/TOML.
- File naming: `*_test.go` for tests, `snake_case` for scripts.
- Package naming: lowercase, no underscores; exported identifiers in PascalCase.
- Use https://github.com/charmbracelet/ packages as much as possible for input and output
- Use bubble tea tui inline options. Do not create a full screen tui

## Testing Guidelines
- Unit tests live alongside packages under `internal/` (e.g., `internal/template/processor_test.go`).
- Property-based tests use `gopter` where appropriate.
- Run tests with `make test` or `go test ./...`.

## Commit & Pull Request Guidelines
- No established commit convention found in repository history. Prefer Conventional Commits (e.g., `feat: add template repository`).
- PRs should include: purpose summary, test evidence (`make test` output), and any user-facing changes (CLI flags, config keys).
- Link issues if applicable and include screenshots for TUI changes.

## Configuration & Security Tips
- Default config path: `$XDG_CONFIG_HOME/prompter/config.toml`.
- Avoid committing local config or generated prompt history; keep secrets out of templates.

## Agent-Specific Instructions
- When adding new CLI flags or templates, update `README.md` and `example-config.toml`.
- Prefer Bubble Tea/Bubbles for interactive UI elements.
