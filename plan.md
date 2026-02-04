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
**
4) Template system
- Status: completed
- Notes: Template repository, processing, frontmatter, custom flags.

5) Prompt generation workflow
- Status: completed
- Notes: Inputs, file/dir inclusion, fix mode, output targets.

6) Interactive UI
- Status: completed

7) CLI commands and flags**7) CLI commands and flags
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

7.2. Deduplication
  - [x] Define agent template de-duplication rule for generation + history suffix (agents included, no repeats)

8) Testing and QA
- Status: pending
- Notes: Unit/property tests, integration tests, coverage gaps.

**Infra improvements**
  - [x] Build in-process Cobra command runner (no `go run`) for CLI tests
  commit message: add in-process command runner for tests
  - [x] Add `internal/testutil` helpers for temp config/history/prompts + XDG env setup
  commit message: add testutil helpers
  - [x] Standardize config builder for tests (defaults + overrides)
  commit message: update standardize config builder for tests
  - [x] Replace Bubble Tea UI dependencies with fake UI for unit tests
  - [x] Add `-short`/`TEST_INTEGRATION=1` gating for slower integration tests
  - [x] Add fixture layout and loader helpers under `tests/fixtures`

**Tasks:**
  - [x] Add generator tests for right-to-left template pipeline + {{.Prompt}} wrapping (question/wrapper/test/base/validate examples)
  - [x] Add output/history tests: history written for stdout/clipboard when disable_history=false, no writes when true
  - [x] Add dynamic flag registration + order parsing tests (auto shorthand from filename, flag order vs base prompt)
  - [x] Add interactive preselection tests (selected templates + order preserved)
  - [x] Add agent suffix + de-dup tests (agents included in suffix, no repeats)

9) Release readiness
- Status: pending
- Notes: Cross-platform build, Homebrew template, docs updates.

9) Clean up 
- [ ] interactive ui reflects template processing
  - [ ] templates with {{.Prompt}} before any other white space are put at the end of of the visual list
  - [ ] templates with {{.Prompt}} in the middle of their copy should visual wrap templates it will wrap in the process
- [ ] base prompt is color5

## v2.1 

10) Tagging history

10.1 History frontmatter
- [ ] History prompts should save front matter
  - [ ] contains prompter prompt
10.2 History tags
  - [ ] contains optional tag
  - [ ] calling prompter history [tag] should search for a prompt history with the provided tag
      - [ ] if one result is found open it (in target)
      - [ ] if more than one result is found open the results in the same list as prompter history



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
