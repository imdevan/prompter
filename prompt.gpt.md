# Goal

Build a Go CLI named `prompter` that assembles high quality prompts for AI coding agents from a base prompt plus optional pre and post templates and contextual information such as files, directories, and captured command output. The tool should be suitable for distribution via Homebrew.

---

## Core Concept

`prompter` generates a final prompt by combining:

- A base prompt provided by the user
- Optional pre and post prompt templates
- Optional contextual information from files, directories, or captured terminal output
- Optional variable substitution via templating

The final prompt can be copied to the clipboard, printed to stdout, written to a file, or opened in an editor.

---

## Distribution

- Build a single static binary named `prompter`
- Provide instructions and assets suitable for a Homebrew tap and formula
- Releases should include macOS and Linux binaries

---

## CLI Usage

### Examples

```sh
prompter "Fix this bug"
prompter -p "No Code" -a "Ask For Tests" -d "Explain this project"
prompter -f main.go -f README.md "Explain the architecture"
prompter --fix
````

---

## Flags

* `--config, -c` string
  Path to config file. Default: `~/.config/prompter/config.toml`

* `--target, -t` string
  Output target. One of:

  * `clipboard` (default)
  * `stdout`
  * `file:/absolute/or/relative/path`

* `--editor, -e`
  Open the composed prompt in an editor after generation

* `--pre, -p` string
  Name of a pre template to prepend. Matches markdown files in `prompts/pre/*.md` by filename stem, case insensitive

* `--post, -a` string
  Name of a post template to append. Matches markdown files in `prompts/post/*.md`

* `--directory, -d` string (optional)
  Include a directory in the prompt. If provided without a value, include the current working directory

* `--file, -f` string (repeatable)
  Include one or more specific files

* `--fix, -x`
  Generate a fix prompt using previously captured terminal command and output

* `--fix-file` string
  Path to captured fix content. Defaults to config value or `/tmp/prompter-fix.txt`

* `--yes, -y`
  Noninteractive mode. Accept defaults and do not prompt the user

---

## Positional Arguments

* Base prompt text
* If omitted:

  * Interactive mode prompts for it
  * Noninteractive mode exits with an error

---

## Interactive Flow (unless --yes)

1. Prompt for base prompt if not provided
2. Prompt to select a pre template if not provided

   * List markdown files in `prompts/pre`
   * Default is any file ending in `.default.md`
   * Option to select None
3. Prompt to select a post template if not provided

   * List markdown files in `prompts/post`
   * Default is any file ending in `.default.md`
   * Option to select None
4. Ask whether to include a directory if not already specified
5. Show a summary and confirm
6. Generate and output the prompt

---

## Fix Mode (--fix)

### Technical Constraint

The CLI cannot automatically access the previous terminal command or output. Fix mode operates only on content that has been explicitly captured and written to a file.

### Intended UX

Users capture failing commands using a shell helper and then run `prompter --fix`.

Example shell helper for bash or zsh:

```sh
fix() {
  local tmp="/tmp/prompter-fix.txt"
  echo "$ $*" > "$tmp"
  echo "" >> "$tmp"
  "$@" 2>&1 | tee -a "$tmp"
}
```

Usage:

```sh
fix go test ./...
prompter --fix
```

### Fix Mode Behavior

* Ignore positional base prompt
* Load fix content from:

  1. `--fix-file` if provided
  2. `$PROMPTER_FIX_FILE` environment variable
  3. Config value `fix_file`
  4. Default `/tmp/prompter-fix.txt`
* If no fix content exists, exit with a clear error explaining how to capture it

### Fix Prompt Construction

1. Pre template

   * Use `prompts/pre/fix.md` if present
   * Otherwise use:

     ```
     Please fix the following issue.
     ```

2. Captured command and output

   * Wrapped in a fenced markdown block labeled as text

3. Optional file and directory context

4. Optional post template

### Example Output

````md
Please fix the following issue.

### Command and output

```text
$ go test ./...
--- FAIL: TestFoo
panic: runtime error: invalid memory address
````

### Relevant files

#### foo.go

```go
func Foo() {
  var x *int
  fmt.Println(*x)
}
```

````

---

## Output Behavior

- `clipboard`
  - Copy final prompt to clipboard
  - Print a confirmation message

- `stdout`
  - Print prompt to stdout

- `file:/path`
  - Write prompt to file
  - Print resolved absolute path

- If `--editor` is set:
  - Open the prompt in a temp file in the resolved editor
  - Editor resolution order:
    1. `--editor` value if string
    2. `$VISUAL`
    3. `$EDITOR`
    4. Config `editor`
    5. `nvim`
    6. `vi`

---

## Configuration

### Format

- Use TOML
- Load via Viper
- Default path: `~/.config/prompter/config.toml`
- Environment variable prefix: `PROMPTER_`
- Precedence: flags > env > config > defaults

### Example config.toml

```toml
prompts_location = "~/.config/prompter"
editor = "nvim"

default_pre = "Engineering Defaults"
default_post = "Ask For Tests"

fix_file = "/tmp/prompter-fix.txt"

max_file_size_bytes = 65536
max_total_bytes = 262144
allow_oversize = false

directory_strategy = "git"
target = "clipboard"
````

### Expected Folder Layout

```
~/.config/prompter/
  config.toml
  prompts/
    pre/
      fix.md
      no-code.default.md
      engineering-defaults.md
    post/
      ask-for-tests.default.md
      summarize-changes.md
```

---

## Directory and File Inclusion Rules

* If inside a git repository:

  * Prefer `git ls-files`
* Otherwise:

  * Walk directory and respect `.gitignore` and `.ignore`
* Exclude binary files
* Exclude files larger than `max_file_size_bytes`
* Cap total included content at `max_total_bytes`
* If limits are exceeded:

  * Include file list and truncated snippets
* Detect language by file extension for code fences

---

## Template System

* Use Go `text/template` with sprig functions
* Delimiters: `{{` and `}}`

### Provided Template Variables

* `.Prompt` string
* `.Now` time
* `.CWD` string
* `.Files` array of:

  * `Path`
  * `RelPath`
  * `Language`
  * `Content`
* `.Git`

  * `Root`
  * `Branch`
  * `Commit`
  * `Dirty`
* `.Config`
* `.Env`
* `.Fix`

  * `Enabled` bool
  * `Raw` string
  * `Command` string
  * `Output` string

### Helper Functions

* `truncate s n`
* `mdFence lang s`
* `indent n s`
* `dedent s`

---

## Noninteractive Mode (--yes)

* Do not prompt
* Use defaults from config
* In fix mode:

  * Require fix content
  * Fail fast if missing
* Respect size limits
* Exit nonzero on errors

---

## Testing Expectations

* Unit tests for:

  * Config precedence
  * Template resolution
  * Fix mode behavior
  * File and directory filtering
* Golden tests for full prompt output

---

## Implementation Hints

* Cobra for CLI
* Viper for config
* Clipboard: atotto/clipboard
* Interactive prompts: survey/v2
* Ignore handling: go-gitignore and git ls-files
* Templates: text/template with sprig

---

## Explicit Non-Goals

* Do not claim access to uncaptured terminal history
* Do not rely on terminal emulator APIs
* Do not silently include large or binary files
