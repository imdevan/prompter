I want to create a prompt to create a go cobra cli tool. 

Please help me evaluate the following promprt to get the most out of my ai agent.

I want to create a Go Cobra app called "prompter"

The app should funciton as a cli tool that will be uploaded to homebrew for consumption. 

The app will take a prompt and set of flags and generate a prompt that can be used 
by an ai coding agent. 

Uses a config file that can be customized and by default is found in ~/.config/prompter

```example
prompter "Fix this bug"
```

Cli Options:

--config, -c: location of config file (checks ~/.config/prompter by default)
--target, -t: target location (defaults to copy prompt to clipboard)
--editor, -e: opens prompt in editor

--pre, -p: pre prompt to prepend 
--post, -o: post prompt to append

--directory, -d: include the current directory in the prompt
--file, -f: include a particular file in the prompt

--yes, -y: auto accept all prmopts, use defaults if available, othwise no pre or post pompts are added. 


Output: 

The cli tool will prompt the user for several inputs
- pre script to use 
  - defaults to file with .default.md ending otherwise defaults to None
  - shows a list of markdown files available in prompts/pre
  - names should be formated in title case (e.g. no-code.md -> No Code)
- post script to use 
  - defaults to file with .default.md ending otherwise defaults to None
  - shows a list of markdown files available in prompts/pre
  - names should be formated in title case (e.g. no-code.md -> No Code)
- include directory? (Y/n) 

After completing prompts the output should confirm the prompt was copied 
to the clipboard, or if the -e flag was passed open the prompt in the editor.

Config file:

What is the best format for a config file? 

Config options:

prompts-location: defaults to ~/.config/prompter
editor: editor to use to dedit, defaults to nvim


Additional considerations:

I want to be able to use some sort of variable insertion templating. 
How would you recommend implementing this feature?




