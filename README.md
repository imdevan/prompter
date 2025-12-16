# Prompter CLI

A Go-based command-line tool that assembles high-quality prompts for AI coding agents by combining base prompts with optional pre/post templates and contextual information from files, directories, and captured command output.

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