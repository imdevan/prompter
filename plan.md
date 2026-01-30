# Development Plan

Status key: pending, in-progress, blocked, completed

## Milestones
1) Confirm scope and requirements
- Status: completed
- Notes: v2_Prompt.md is authoritative; UI uses Lip Gloss for layouts with Bubble Tea/Bubbles; config precedence local -> global -> defaults; include_agents default "all"; output targets include named "editor".

2) Repo scaffolding
- Status: completed
- Notes: Go module initialized; directories and placeholder files created; Makefile, scripts, and CI placeholders added.

3) Configuration and path resolution
- Status: completed
- Notes: Config manager, defaults, precedence, validation.

4) Template system
- Status: completed
- Notes: Template repository, processing, frontmatter, custom flags.

5) Prompt generation workflow
- Status: pending
- Notes: Inputs, file/dir inclusion, fix mode, output targets.

6) Interactive UI
- Status: pending
- Notes: Bubble Tea/Bubbles prompts and selection lists.

7) CLI commands and flags
- Status: pending
- Notes: Cobra commands, completion, version, help text.

8) Testing and QA
- Status: pending
- Notes: Unit/property tests, integration tests, coverage gaps.

9) Release readiness
- Status: pending
- Notes: Cross-platform build, Homebrew template, docs updates.

## Progress Log
- 2025-01-29: Created plan.md with milestones and status tracking.
- 2025-01-29: Confirmed scope with UI/layout expectations and default include_agents behavior.
- 2025-01-29: Scaffolded repository structure and baseline files.
- 2025-01-29: Started configuration and path resolution implementation.
- 2025-01-29: Implemented config defaults, XDG path helpers, and TOML loading with precedence.
- 2025-01-29: Started template system implementation.
- 2025-01-29: Implemented filesystem template repository, frontmatter parsing, and template rendering helpers.
