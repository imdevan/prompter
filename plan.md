# Development Plan

Status key: pending, in-progress, blocked, completed

## Milestones
1) Confirm scope and requirements
- Status: completed
- Notes: source_of_truth.md is authoritative; Bubble Tea/Bubbles inline UI; prefer green/pink; config precedence local -> global -> defaults; include_agents default "all"; output targets include "editor".

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
- Status: completed
- Notes: Inputs, file/dir inclusion, fix mode, output targets.

6) Interactive UI
- Status: completed
- Notes: Bubble Tea/Bubbles prompts and selection lists.

7) CLI commands and flags
- Status: in-progress
- Notes: Cobra commands, completion, version, help text, dynamic template flags from config.

### Dynamic template flag registration
- [x] load templates + frontmatter during config init to derive flags/shorthands
- [x] register cobra flags before command parsing
- [x] map flag usage to template selection order in request

### Subcommands
- [x] config
  - [x] no args: open config location in editor
  - [x] init  
  flags:
    - [x] editor, e - open in editor
    - [x] force, f - replaces existing template

- [x] add
  - [x] adds template to prompts_location
  flags:
    - [x] editor, e - open in editor
    - [x] force, f - replaces existing template
    - [x] interactive, i - force interactive

- [x] history
  - [x] no args opens history in editor
  - [x] number provided: output the generated prompt however many steps back

- [x] fix
flags:
  - [x] editor, e - open in editor
  - [x] interactive, i - force interactive
  - [x] yes, y - force non interactive

- [x] list
  - [x] list available templates use bubbles and lip gloss

- [x] edit
  - [x] no args: open prompts_location in editor
  - [x] 1 arg: edits template in prompts_location
  - [x] if not found prompt user to add using bubbles confirm

- [x] completion
  - [x] generate shell completion scripts via cobra

## Queue


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
- 2025-01-29: Started prompt generation workflow implementation.
- 2025-01-29: Completed prompt generation workflow and started interactive UI work.
- 2025-01-29: Implemented Bubble Tea/Bubbles interactive base prompt and template selection UI.
- 2025-01-29: Started CLI wiring with Cobra root command and generate flow.
