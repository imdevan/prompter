1.1 see it in action

header

```bash
 prompter

 Base prompt
 Enter your base prompt
 ╭─────────────────────────────────────────────────────╮
 │ ┃ My prompt                                         │
 │ ┃                                                   │
 │ ┃                                                   │
 ╰─────────────────────────────────────────────────────╯
 Press Enter to continue. Alt+Enter for a new line.

 ╭─────────────────────────────────────────────────────╮
 │                                                     │
 │  ╭──────────╮╭───────────────────╮╭─────────────╮   │
 │  │ add-test ││ stick_to_the_plan ││ "My prompt" │   │
 │  ╰──────────╯╰───────────────────╯╰─────────────╯   │
 │                                                     │
 │    Select templates                                 │
 │                                                     │
 │  │ [x] Add Test                                    │
 │  │     Generate tests based on context              │
 │                                                     │
 │    [ ] Question                                    │
 │        Do not generate code                         │
 │                                                     │
 │    [x] Next Task                                    │
 │        Do the next task in plan.md, only.           │
 │                                                     │
 │    [ ] Keymap                                       │
 │        Create nvim keymap                           │
 │                                                     │
 │                                                     │
 │                                                     │
 │                                                     │
 │                                                     │
 │                                                     │
 │    ••                                               │
 │                                                     │
 │    󱁐 Toggle • 󰌑 Continue • / filter • ? more        │
 │                                                     │
 ╰─────────────────────────────────────────────────────╯


 ╭─────────────────────────────╮
 │                             │
 │    Copied to clipboard  󱁖  │
 │                             │
 ╰─────────────────────────────╯

```

1.2 bug fixes

```
bun test | prompter --fix

```

1.3 Skills compatible

```
Global templates
~/.local/share/prompter/prompts

  Add Test  
  Generate tests based on context
  -a, --add-test                 

  Question  
  Do not generate code
  -q, --question      

  Next Task
  Do the next task in plan.md, only.
  -s, --stick-to-the-plan           

  Keymap
  Create nvim keymap  
  -v, --vnk-neovimkeys

Global Skills

  Frontend Design
  From ~/.opencode/skills
  -f, --frontend-design

  Summarize Context
  From ~/.opencode/skills
  -u, --summarize      

Local Skills

  Local Skill Example
  From .agents/skills
  -l, --local-skill  

  custom-cursor-command
  From .cursor/commands      
  -c, --custom-cursor-command

Agent templates

  Agent instructions
  From AGENTS.md 
  -a, --agents-md
```

