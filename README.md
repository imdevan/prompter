# prompter-cli

Prompter assembles high-quality prompts for AI coding agents by combining base prompts with templates and contextual data from files, directories, and command output.

## Quick start
```bash
prompter "explain this output"
```

## Commands
```bash
prompter list
prompter list -a # include agent templates (AGENTS.md, .cursor/commands, .kiro/steering, opencode)
prompter --directory # include current directory path in prompt
prompter --dir-content # include current directory content in prompt
```

## Development
```bash
just build
just test
```

See `INSTALL.md` for installation options and `example-config.toml` for configuration.
