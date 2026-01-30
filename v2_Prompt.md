# Prompt to Recreate Prompter CLI

Create a Go-based command-line tool called "prompter" that assembles high-quality prompts for AI coding agents by combining base prompts with templates and contextual information from files, directories, and captured command output.

## Project Overview

**Name**: prompter-cli
**Module**: prompter-cli
**Language**: Go

## Core Functionality

The tool should generate a prompt based on user input in 1 of 2 modes

Both modes should:

1. Generate a prompt based on user input and selected template files
2. Include contextual information from files, directories, git, and clipboard
3. Accept flags
4. Accept piped in values
5. Optionally save prompts in history_location
6. Output the assembled prompt to clipboard, stdout, or editor

### Interactive

1. Allow users to enter a base prompt (interactively, via argument, or from clipboard)
2. Allow user to recursively select templates to use

### Non-interactive

1. all values are passed in at call time. 

## Additional Commands

Manage prompt templates (add, list, fix, open in editor, hist)
Config management (config, config init)

## Style

bubbletea + bubbles by charmbracelet for ui output and interactive inputs
prefer green and pink colors

All commands should be output via bubbles e.g. bubbletea and all inputs should use bubbles input options

## Project Structure

```bash
prompter-cli/
├── cmd/
│   └── prompter/
│       ├── main.go              # Main CLI entry and execution
│       ├── root.go              # Root command, global flags, and help
│       ├── generate.go          # Triggers prompt generation workflow
│       ├── add.go               # Triggers template creation workflow
│       ├── list.go              # Triggers template listing workflow
│       ├── edit.go              # Triggers template editing workflow
│       ├── config.go            # Opens or inspects the config file
│       └── config_init.go       # Generates a default config file (prompter config init / --init)
│
├── internal/
│   ├── domain/
│   │   ├── request.go           # Defines what a prompt generation request is and validates it
│   │   ├── template.go          # Defines what a template is and its invariants
│   │   ├── config.go            # Defines config shape and DefaultConfig()
│   │   └── errors.go            # Domain level error types and rules
│   │
│   ├── app/
│   │   ├── app.go               # Wires dependencies and exposes workflows to the CLI
│   │   └── app_test.go
│   │
│   ├── workflow/
│   │   ├── workflow.go          # Holds shared dependencies and constructors
│   │   ├── generate.go          # Implements the prompt generation flow
│   │   ├── add.go               # Implements template creation and persistence
│   │   ├── list.go              # Implements template discovery and formatting
│   │   ├── edit.go              # Implements editor launching behavior
│   │   ├── config.go            # Implements config viewing and editing behavior
│   │   ├── config_init.go       # Implements default config creation logic
│   │   ├── errors.go            # Defines workflow specific error mappings
│   │   └── workflow_test.go
│   │
│   ├── template/
│   │   ├── processor.go         # Applies values to a domain.Template
│   │   ├── repository.go        # Loads and saves templates from storage
│   │   ├── interface.go         # Describes how templates are retrieved and persisted
│   │   └── processor_test.go
│   │
│   ├── config/
│   │   ├── manager.go           # Reads, writes, and checks existence of config files
│   │   ├── interface.go         # Describes how config persistence behaves
│   │   └── manager_test.go
│   │
│   ├── interactive/
│   │   ├── prompter.go          # Collects user input and builds a domain.Request
│   │   └── prompter_test.go
│   │
│   ├── output/
│   │   ├── handler.go           # Formats and delivers generated output
│   │   ├── interface.go         # Describes how output destinations behave
│   │   └── handler_test.go
│   │
│   ├── adapters/
│   │   ├── bubbletea/
│   │   │   ├── prompter.go      # Wraps the TUI library behind a simple input API
│   │   │   └── prompter_test.go
│   │   │
│   │   ├── clipboard/
│   │   │   ├── clipboard.go     # Wraps clipboard access behind a stable interface
│   │   │   └── clipboard_test.go
│   │   │
│   │   └── editor/
│   │       ├── editor.go        # Wraps OS process execution for launching editors
│   │       └── editor_test.go
│   │
│   ├── utils/
│   │   ├── fs.go                # Simplifies file and directory operations
│   │   ├── paths.go             # Resolves configuration and template locations
│   │   ├── strings.go           # Normalizes and formats text values
│   │   └── utils_test.go
│   │
│   └── errors/
│       └── errors.go            # Holds shared infrastructure error types
│
├── scripts/
│   ├── build.sh                 # Builds binaries for multiple platforms
│   └── release.sh               # Packages and publishes releases
│
├── homebrew/
│   └── prompter.rb.template     # Template for generating a Homebrew formula
│
├── .github/
│   └── workflows/
│       └── release.yml          # Automates building and publishing releases
│
├── example-config.toml          # Demonstrates a valid configuration file
├── INSTALL.md                   # Explains installation methods
├── README.md                    # Explains purpose, usage, and design
├── Makefile                     # Defines local development tasks
├── go.mod
└── go.sum
```

## Dependencies

**Direct dependencies:**
- `github.com/spf13/cobra` 
- `github.com/leanovate/gopter` 
- Interactive prompts:
- https://github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/bubbles

**Indirect dependencies (via viper and other packages):**
- `github.com/spf13/viper` - Configuration management
- `github.com/atotto/clipboard` - Clipboard operations
- `github.com/Masterminds/sprig` - Template functions
- `github.com/pelletier/go-toml` - TOML parsing

## Configuration System

**Default config location**: `$XDG_CONFIG_HOME/prompter/config.toml`

**Configuration options:**
- `prompts_location` - Where prompt templates are stored (default: `$XDG_DATA_HOME/prompter/prompts`) 
- `history_location` - Where old prompts are stored, or if prompt is opened with editor flag (default: `$XDG_CACHE_HOME/prompter/history`)
- history_clear_cycle - 'never', or number of days
- history_file_format - What to save the prompts as in history default 'month-day_eu time with 0s in the tens place for 0-9'
- `local_prompts_location` - Local prompts location relative to CWD (default: `prompts`) 
- include_agents - 'all', 'agents.md', 'cursor', 'kiro', opencode (global) can be one or a combination
- `editor` - Default editor for opening prompts default nvim
- `directory_strategy` - How to include directories: "git" or "filesystem"
- `target` - Default output target: "clipboard", "stdout", or "file:/path" defaults to "clipboard"
- `interactive_default` - Default interactive mode (true/false)https://github.com/charmbracelet/bubbles
- `include_builtin_shorthand` - (true/false) if false removes the shorthand flags for builtin flags to accommodate space for more template shorthand flags
- `remap_short_flags` - this should be a section of the config that allows the user to remap the shorthand flags of builtin flag functions

**Configuration precedence:**
1. Local config file (in current directory)
2. Global config file (`kyh,m,jm.nikyujhmn,b iojklm,n l;p/., ~/.config/prompter/config.toml` or `~/.prompter/config.toml`)
3. Default values

## Template System

**Template locations:**
1. Local prompts directory (current working directory + `prompts/` or configured `local_prompts_location`)
2. Global prompts directory (`~/.config/prompter/prompts` or configured `prompts_location`)
3. Custom template locations (from config)

4. treat agents.md (local), .cursor/commands, and .kiro/steering as templates if --agents flag is true

**Template structure:**
- Templates are stored as `.md` files
- `Template location{prompts_location}/{name}.md`
- Special templates: 
  - `{location}/index.md` included before any other flags or templates, included by default
  - `{location}/fix.md` (used in fix mode) [questioning]

**Template processing:**
- Uses Go text/template with sprig functions
- Core template variables:
  - `.Prompt` - The prompt as it has been assembled so far
  - `.BasePrompt` - The user's base prompt
  - `.Files` - Array of included file contents
  - `.Directory` - Directory information
  - `.FixContent` - Command output (in fix mode)

- Additional variables

  - `.CWD` - current working directory
  - `.Git` - contains information about git repo, if called from git repo
    - example:
    ```
    {{- if .Git.Root }}
    Repository: {{ .Git.Root | base }}
    Branch: {{ .Git.Branch }}
    Commit: {{ .Git.Commit | truncate 8 }}
    {{- if .Git.Dirty }}⚠️ Uncommitted changes{{- end }}
    {{- end }}
    ```
  - `.fix` - variable present if called from --fix mode
    - example:
    ```
    {{- if .Fix.Enabled }}
    Command: {{ .Fix.Command }}
    Output: {{ .Fix.Output }}
    {{- end }}
    ```
  - `.Env` - contains environment variables if present
    - example:
    ```
    User: {{ .Env.USER }}
    Editor: {{ .Config.Editor }}
    ```
**Sprig function examples**

validate that the following configurations work
functions

```
# String manipulation
{{ .Prompt | truncate 50 }}
{{ .Text | indent 4 }}
{{ .CWD | base }}

# Code fence
{{ mdFence "go" .Code }}

# Date formatting
{{ .Now.Format "2006-01-02 15:04:05" }}
```

Control structures
```
{{- if .Git.Root }}
In git repo
{{- else }}
Not in git repo
{{- end }}

{{- with .Git.Branch }}
Branch: {{ . }}
{{- end }}
```



**Template Frontmatter:**

Templates may optionally contain front matter which can contain the following options

The frontmatter is not added to the prompt as a part of the template. 

```bash
title: string # template title when showing in interactive mode and help menu
description: string # description in interactive list and help menu
flag: string # custom flag 
shorthand: string # custom shorthand flag
pin: true | false # if pin is true the template will be put at the top of the list in inquery'explain this test output''explain this test output'inquery'explain this test output''explain this test output'interactive mode
```

## CLI Commands

### Main Command: `prompter [base-prompt]`
Assembles and outputs a proinquery'explain this test output''explain this test output'mpt.

Base prompt can be passed as a string or piped in via `echo "[base-prompt]" | prompter`

**Built in Flags:**
- `--config string` - Config file path (default: `~/.config/prompter/config.toml`)
- `--file strings` - Files to include
- `-d, --directory` - Include current directory
- `-t, --target string` - Output target (clipboard, stdout, file:/path)
- `-f, --fix` - Fix mode - process captured command output
- `-b, --clipboard` - Append clipboard content to prompt (or use as base prompt if none provided)
- `-i, --interactivinquery'explain this test output''explain this test output'inquery'explain this test output''explain this test output'e` - Force interactive mode (overrides config default) 
- `-y, --yes` - Non-interactive mode - use defaults without prompts
- `-v, --version` - Print version information
- Custom template flags (dynamically registered from config)

**Interactive mode:**
- Prompts for base prompt if not provided - if a user hits enter continue to generating with templates and empty base prompt string
- Prompts for template selection (with "None" option) via searchable input with bubbles
- Recursively asks for template input until user selects none or done if it is easy to change the string
- Shows templates from all locations (local, global, custom) with labels

**UI**
- All ui elements should be generated with bubbles / bubbletea as an inline TUI (NOT FULLSCREEN)

**Piping in values**

- If a value is piped in e.g. `bun test | prompter 'explain this test output'` treat the piped in value as if it were the last block of the prompt template process.

examples:

```
bun test | prompter 'explain this test output'
> bubbles viewport of long running process
> "Prompt copied to clipboard! 🎉"
"
explain this test

<Test output>
"

# elsewhere in template quality_assurance.md
"
# Goal

Evaluate this test output. How can these tests be improved from a qa perspective? 
"

bun test | prompter -q
> bubbles viewport and stopwatch if long running process
> "Prompt copied to clipboard! 🎉"
"
# Goal

Evaluate this test output. How can these tests be improved from a qa perspective? 

<Test output>
"

```

### Subcommands:

#### `prompter fix`
- calls the previously executed shell command and pipes the result into prompter
- similar to calling `npm test | prompter 'please fix'` but the system should retrieve the previous command from the terminal history
- Load `fix.md` template from prompts location (if exists) otherwise default to 'Please evaluate the following output for errors, and resolve an issues. If there are multiple issues please fix them in order.'

**examples:**

```bash
> bun test
... failing test ...

prompter fix
> bubbles viewport and stopwatch if long running process
> "Prompt copied to clipboard! 🎉"
```
#### `prompter list`

- Lists all available templates from all locations
- Groups templates by local, global, or custom
- Use bubletea output to make it look pretty. 

#### `prompter add [content]`


- accepts optionally up to two args
  - arg 1: name of prompt template to add
  - arg 2: template content
- Adds a new prompt template
- with basic default frontmatter
- Flags:
  - `-b, --clipboard` - Create template from clipboard content
  - `-r, --overwrite` - Overwrite existing template without prompting
  - `-e, --edit` - Open prompt in editor after creation




#### `prompter prompts`

- Opens the prompts directory in the configured editor

#### `prompter version`


- Prints version information (version, commit, build date, Go version, platform)

#### `prompter completion [shell]`
- Generates shell completion scripts
- should primarily come from cobra

## Prompt Generation Flow

1. **Load Configuration**: Load and resolve config (local + global precedence)
2. **Resolve Interactive Mode**: Based on flags (`-i`, `-y`) or config default
3. **Collect Inputs** (if interactive):
   - Base prompt (if not provided)
   - Template selection
4. **Collect Contentenable")**:
   - Base prompt (from arg, clipboard, or interactive)
   - Files (read and include)
   - Directory (if `--directory` flag, include current directory)
   - Fix content (if fix mode, read from fix_file)
5. **Load Templates**:
   - Load pre-template if specified
   - Load post-template if specified
   - Load fix template if in fix mode
6. **Process Templates**:
   - Process template with context data
   - Flags and arguments are processed in the order in which they are received. prompter --template --template2 [base-prompt] is different than prompter --template [base-prompt ]--template2 

7. **Output Prompt**:
all show atheistically pleasing output via bubbletea+bubbles+lipgloss

   - To clipboard (default) Still shows atheistically pleasing output via bubbletea+bubbles+lipgloss
   - To stdout
   - file - outputs to file location if provided
   - To editor - opens prompt in prompt_history locaion in configed editor

## Directory Inclusion

When `--directory` flag is used:
- **Git strategy** (default): Include only files tracked by git
- **Filesystem strategy**: Include all files in directory (respecting .gitignore)

## Output Targets

- **clipboard** (default): Copy prompt to system clipboard
- **stdout**: Print prompt to standard output
- **file:/path**: Write prompt to file at specified path
- **editor**: Open prompt in configured editor (or EDITOR/VISUAL env var)

## Build System

**Makefile targets:**
- `make build` - Build for current platform
- `make dev-build` - Development build (no optimization)
- `make cross-platform` - Build for all platforms (uses `scripts/build.sh`)
- `make install` - Install to `/usr/local/bin`
- `make test` - Run tests
- `make clean` - Clean build artifacts

**Build-time variables** (injected via ldflags):
- `version` - Version string
- `commit` - Git commit hash
- `date` - Build date

**Cross-platform builds:**
- macOS (Intel): `prompter-darwin-amd64`
- macOS (Apple Silicon): `prompter-darwin-arm64`
- Linux (Intel): `prompter-linux-amd64`
- Linux (ARM64): `prompter-linux-arm64`
- build for AUR package yay can install

## Testing

- Unit tests for all packages
- Property-based tests using gopter
- Integration tests for template processing
- Test files follow naming: `*_test.go`

## Key Implementation Details

1. **Custom Template Flags**: Dynamically register CLI flags for custom templates defined in config
2. **Template Location Resolution**: Check local directory first, then global, then custom locations
3. **Interactive Mode Resolution**: Flags (`-i`, `-y`) override config default
4. **Clipboard Handling**: Can use clipboard as base prompt or append to existing prompt
5. **File Size Limits**: Configurable max file size and total size limits (with allow_oversize option)
6. **Path Contraction**: Display paths with `~` for home directory in user-facing messages
7. **Template Defaults**: Support `.default.` prefix for default templates
8. **Number Selection**: Optional number key selection for templates in interactive mode

## Example Usage

```bash
# Interactive mode
prompter

# With pre and post templates
prompter --pre question --post clarify "how do I build this project?"

# Include current directory
prompter --directory "analyze this codebase"

# template prompts/analyze.md
prompter -a

# Fix mode
prompter fix

# Add a template
prompter add question "This is a question template"

# List templates
prompter list

# Open prompts directory
prompter prompts
```

## Installation

**Homebrew:**
```bash
brew install imdevan/prompter/prompter
```

**Manual:**
- Download pre-built binaries from releases
- Or build from source: `make build && sudo make install`

## Additional Requirements

- Support for shell completion (bash, zsh, fish, powershell)
- GitHub Actions workflow for releases
- Homebrew formula template
- Comprehensive error handling and user-friendly error messages
- Respect .gitignore when including directories
- Support for both TOML config files
- Template functions from sprig library (string manipulation, date formatting, etc.)
