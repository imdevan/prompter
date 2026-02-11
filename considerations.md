# Changes for consideration

- Change history to hist
- Change package name to (or include alias for) prmpt
  - current: prompter ...
             prompter history
    change: prmpt ...
            prmpt hist

- Access tagged prompts from root
  current: prompter history prompt_tag
  change:  prompter #prompt_tag         # In addition to standard history function

- Nested templates 

```markdown_format.md
Use this markdown format:

# Feature
- [ ] 1 Task
- [ ] 1.1 Sub task
```

```markdown_expander.md
{{.Tempalte "markdown_format"}}

{{.ClipBoard}}
```

usage:
```bash
prompter -m 
```


- Remove remapping builtin shortflags
  - not needed now after changing them to capital letters

- Add shell execution at template generation time?

- Change default locations to ~/.prompter? curren is XDG_ recommended locations
