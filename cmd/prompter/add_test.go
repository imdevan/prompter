package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"prompter-cli/internal/utils"
)

func TestAddCommandWritesTemplate(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))

	promptsDir := filepath.Join(root, "prompts")
	if err := writeConfigWithPrompts(promptsDir); err != nil {
		t.Fatalf("write config: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	opts := &addOptions{}

	args := []string{"question", "Hello {{ .BasePrompt }}"}
	if err := runAdd(cmd, opts, args); err != nil {
		t.Fatalf("run add: %v", err)
	}

	path := filepath.Join(promptsDir, "question.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	if got := string(data); got != "Hello {{ .BasePrompt }}" {
		t.Fatalf("unexpected template content: %q", got)
	}
	if !strings.Contains(buf.String(), "Added template question") {
		t.Fatalf("expected output to include template name, got %q", buf.String())
	}
}

func TestAddCommandRejectsOverwrite(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))

	promptsDir := filepath.Join(root, "prompts")
	if err := writeConfigWithPrompts(promptsDir); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	existingPath := filepath.Join(promptsDir, "question.md")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cmd := &cobra.Command{}
	opts := &addOptions{}
	args := []string{"question", "Hello {{ .BasePrompt }}"}
	if err := runAdd(cmd, opts, args); err == nil {
		t.Fatal("expected error when template already exists")
	}
}

func writeConfigWithPrompts(promptsDir string) error {
	configPath := utils.ConfigPathGlobal()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	content := []string{
		"prompts_location = \"" + promptsDir + "\"",
		"editor = \"nvim\"",
	}
	return os.WriteFile(configPath, []byte(strings.Join(content, "\n")+"\n"), 0o644)
}
