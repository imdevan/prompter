# Docs updates
## Update landing page

- [ ] 1 Header copy changes

- [ ] 2 See it in action section:
  - [ ] 2.1 Remove "From simple prompts to complex workflows, prompter adapts to your needs."
  - [ ] 2.2 basic usage replace with current interactive implementation
bug fixes: change to pipe implementation 
`bun test | prompter -f -y`

- [ ] 3 update everything you need section
  - [ ] 3.1 remove open source card.
  - [ ] 3.2 remove fix mode card
  - [ ] 3.3 add highly customizable
  - [ ] 3.4 add any skill anywhere 

- [ ] 4. Get started section
  - [ ] 4.1 change configure
  `propmter config init`

## Docs:

- [x] 5 Install
- [x] 5.1 add arch install after homebrew

6 Sidebar outline

Intro (current)
Usage Guide
Custom Templates

New side bar:
Prompter Cli / Install 
Commands
  - derive all subcommands from project
    - Add commands include:
      - description
      - command, interactive and non interactive
      - flags
      - examples
Templates
  Prompt Templates
  Discovery
  Index.md
  Template Syntax
Configuration
Nvim integration 
Advanced
  Use cases
  Best practices
  

## Docs refinements

2. probably just regenerate template syntax? examples are bad. 
 available data is the only good part.
 should address rendering order

1 add section  about prompt generation order
  where? (prompter gen, Template Syn)?

2 Neovim integration 
  - save current docs to old
  - replace neovim integration
