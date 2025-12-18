# Prompter CLI

A Go-based command-line tool that assembles high-quality prompts for AI coding agents by combining base prompts with optional pre/post templates and contextual information from files, directories, and captured command output.

## Usage

### Default

```
prompter
```

By default just running prompter will initialize interactive prompt generation.


### Fix mode

```
prompter --fix
```

Will rerun the previous shell command and copy the content to the clipboard.

### Available Commands

Extra helper commands to help manage prompt-templates.

```
add         Add a new prompt template
completion  Generate the autocompletion script for the specified shell
help        Help about any command
list        List available prompt templates
prompts     Open prompts directory in editor
version     Print version information
```

### Flags

Lots of useful flags to add files, current directory, cipboard contents and more.

```
-b, --clipboard         append clipboard content to prompt (or use as base prompt if none provided)
-c, --config string     config file path (default ~/.config/prompter/config.toml)
-d, --directory         include current directory
-e, --editor string     editor to open prompt in
    --file strings      files to include
-f, --fix               fix mode - process captured command output
    --fix-file string   file containing command output to fix (overrides config)
-h, --help              help for prompter
-i, --interactive       force interactive mode (overrides config default)
-n, --numbers           enable number key selection for templates
-o, --post string       post-template name
-p, --pre string        pre-template name
-t, --target string     output target (clipboard, stdout, file:/path)
-v, --version           print version information
-y, --yes               noninteractive mode - use defaults without prompts
```

## Configuration

Prompter by default checks `~/.config/prompter/config.toml` for config options. 

Custom config path can also be set via flag

```
prmopter -c custom-config.toml
```

See [example config](./example-config.toml) for what options are configurable.

## Prompt-Templates

Prompter by default checks for tempaltes in `~/.config/prompter/prompts`, 
but this can be changed with the `prompts_location` in the `config.toml`

Prompter also checks the current local directory for a `prompts`. 
This can be changed in the config with `local_prompts_location`.
If both a local and global prompts are found, prompter will use both. 

Prompt templates are broken up into two seperate categories. 

`pre` templates go before the base_prompt input
`post` templates go after the base_prompt input

### Example

```
prompter --pre question --post clarify --directory "how do I build this project?"
```

will generate the following prompt assuming `prompts_location/pre/question` and `prompts_location/post/clarify` are defined.

```
# Question 

The following prompt is a question do not output any code or artifacts. 

# Prompt
how do I build this project?

Referencing: 
User/usr/.../current-folder

# Clarify
Ask clarifying questions do not jump to the first answer you think of
```

Special case: 

`fix.md` is an optional template that can be saved in the root prompt location
and is used in `prompter --fix` and will prepend the fix template to the previously
executed terminal command. 

## Project Structure

```
prompter-cli/
├── cmd/
│   └── prompter/           # Main CLI application entry point
│       └── main.go
├── internal/
│   ├── app/                # Application orchestration layer
│   │   └── app.go
│   └── interfaces/         # Core interfaces and data structures
│       ├── config.go       # Configuration management interface
│       ├── template.go     # Template processing interface
│       ├── content.go      # Content collection interface
│       ├── output.go       # Output handling interface
│       ├── interfaces_test.go
│       └── property_test.go
├── pkg/
│   └── models/             # Shared data models
│       └── request.go
├── prompts/                # Template directories
│   ├── pre/
│   └── post/
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- **github.com/spf13/cobra** - CLI framework
- **github.com/spf13/viper** - Configuration management
- **github.com/AlecAivazis/survey/v2** - Interactive prompts
- **github.com/atotto/clipboard** - Clipboard operations
- **github.com/Masterminds/sprig/v3** - Template functions
- **github.com/leanovate/gopter** - Property-based testing

## Core Interfaces

### ConfigManager
Handles configuration loading, precedence resolution, and validation.

### TemplateProcessor
Manages template loading, execution, and custom helper functions.

### ContentCollector
Collects and filters content from files and directories.

### OutputHandler
Manages different output destinations (clipboard, stdout, file, editor).

## Building

```bash
go build ./cmd/prompter
```

## Testing

```bash
go test ./...
```

## Usage

```bash
./prompter --help
```
