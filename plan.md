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

4.1 Support opencode compatible skills as templates. 
- [ ] refactor templates title to name
- [ ] Add opencode compatible skills as templates
  - opencode reference https://opencode.ai/docs/skills/
  - enabled if include_agents == all || opencode
  - See placefiles and understand discovery for skills discovery and formats.
    - Template names should be derived from folder name OR. name from frontmatter. 
    - flags and shorthand flags may be defined in metadata

5) Prompt generation workflow
- Status: completed
- Notes: Inputs, file/dir inclusion, fix mode, output targets.

6) Interactive UI
- Status: completed

7) CLI commands and flags7) CLI commands and flags
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

9) Interactive ui updates
- [ ] interactive ui reflects template processing
  - [ ] templates with {{.Prompt}} before any other white space are put at the end of of the visual list
  - [ ] templates with {{.Prompt}} in the middle of their copy should visual wrap templates it will wrap in the process
- [x] base prompt is color3

10) Tagging history
- [ ] History may have frontmatter
10.1 Prompt tag
- [x] Add a tag flag to root command. If present, history should be created with frontmatter and tag populated
10.2 History frontmatter
- [x] calling prompter history [tag] should search for a prompt history with the provided tag
  - [x] if one result is found open it (in target)
  - [x] if more than one result is found open the results in the same list as prompter history

11. History refinements

11.1 history clear subcommand
- [x] add history clear subcommand
  - [x] confirms with user using huh confirm
  - [x] clears the history files
  - [x] keep tags flag to keep all with tags
  - [x] editor, -e flag opens history folder in editor

11.2 style
- [x] display history based on this algorithm
  - continue to use the default delegation function - no need for extra headers
- [x] history_enable_time_ago defaults to true - add to config and config init
- [x] history_date_time format defaults to day, month - add to config and config init

if history_enable_time_ago:
items less then a day old
```
#tag            -- if present
One minute ago  -- in bold use *ago* format if less then a week old
file_name • 0 B -- light font as date is now. prompter- and .md omitted from file name
```


less then a month old
```
#tag            -- if present
Tues. 17th, time      -- in bold use *ago* format if less then a week old
file_name • 0 B -- light font as date is now. prompter- and .md omitted from file name
```


items after a month or if history_enable_time_ago
```
#tag            -- also in bold if present
date_time       -- in bold 
file_name • 0 B -- light font as date is now. prompter- and .md omitted from file name
```

11.3 delete from list
- [x] from history list view. hitting d/D, del, or backspace will remove a history item
  - [x] d/backspace: prompt the user before deleting
    - [x] prompt should show a bubbles viewport of the history file
    - [x] and a confirmation dialog
  - [x] D/del: delete without dialog
  - [x] upon deletion the list should update but user should continue in the history list view focused on the next item in the list
  - [x] keybinding should show "d/D delete"

11.4 cache maintenance
- [x] on history call delete empty prompts
  - [x] also delete any prompts that only contain the index template (also essentially empty)

11.5 insert from list

- [x] from history list hitting i/ins will open the prompt in configured editor with 
  
    ```
    \n
    ---
    \n
    ```
    inserted above the last text in insert mode above that
- [x] or if history called with flag -n, --insert

11.6 Improve keybinding tips. 
  - [x] adjust visible keybind help in ui.list (both prompt select templates and hist)
    - prevent tips from forcing width open. 
    - show all keybinds in ? menu
                        
12 Config update
12.1 Color scheme change
- [x]replace colors with colors derived from config primary, secondary, accent base_prompt, border
  - the colors default to their current color definitions
- [x] add to config defaults and init
e.g. `primary=7 or other color type that lipgloss supports`

13 Escape
- [x] quiting a bubble tea program should end without a prompt being generated
  only generate a prompt when a user completes the input flow. this applies to all bubble tea inputs


14) color scheme changes
- [x] update color scheme

15.00 Improve editor / vim integration
- [x] 15.1 on root command if in editor insert cursor at end of of file
- [x] 15.2 define prompt separator in config defaults to '---'
- [x] 15.3 if target ==== clipboard and --editor ==== true
  on root command
    - [x] when vim closes copy all content after frontmatter to the clipboard
  on insert from history
    - [x] copy all content between frontmatter and last prompt seperator ---


16) Release readiness
 Status: pending
- Notes: Cross-platform build, Homebrew template, docs updates.

16.1 config init changes
- [x] should comment out all default options (everything)

- [ ] 16.2 build for arch
- [ ] 16.3 build for homebrew


## V2.1

- [ ] create a ui util to manage responsiveness. it should track the width of the view and 
  - [ ] render which tips are visible based on ui width
  - [ ] only render borders in md or larger views

sm: < 40
md: < 80
lg: > 80
xl: > 300
