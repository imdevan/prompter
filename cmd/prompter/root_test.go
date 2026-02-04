package main

import (
	"testing"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/testutil"
)

func TestRegisterTemplateFlagsUsesFilenameDefaults(t *testing.T) {
	testutil.WithTempXDG(t)
	cfg := domain.DefaultConfig()
	opts := &rootOptions{}
	cmd := newRootCmd()

	templates := []domain.Template{
		{Name: "question"},
		{Name: "test"},
	}

	registerTemplateFlags(cmd, opts, cfg, templates)

	if cmd.Flags().Lookup("question") == nil {
		t.Fatalf("expected --question flag to be registered")
	}
	if cmd.Flags().Lookup("test") == nil {
		t.Fatalf("expected --test flag to be registered")
	}
	if cmd.Flags().Lookup("question").Shorthand != "q" {
		t.Fatalf("expected shorthand -q, got %q", cmd.Flags().Lookup("question").Shorthand)
	}
}

func TestResolveTemplateOrderPreservesBasePromptPosition(t *testing.T) {
	cfg := domain.DefaultConfig()
	opts := &rootOptions{
		templateFlagName: map[string]string{
			"question": "question",
			"test":     "test",
		},
		templateFlagShorthand: map[string]string{
			"q": "question",
			"t": "test",
		},
		templateShortByName: map[string]string{
			"question": "q",
			"test":     "t",
		},
	}

	args := []string{"-q", "-t", "base prompt", "-t"}
	templates, baseIndex, _ := resolveTemplateOrder(args, opts, cfg)

	if baseIndex != 2 {
		t.Fatalf("expected base prompt index 2, got %d", baseIndex)
	}
	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}
	if templates[0] != "question" || templates[1] != "test" || templates[2] != "test" {
		t.Fatalf("unexpected template order: %v", templates)
	}
}

func TestTemplateShorthandsIncludeAgentsOnce(t *testing.T) {
	names := []string{"agents.md", "agents.md", "question"}
	mapping := map[string]string{"question": "q"}
	shorts := templateShorthandsForNames(names, mapping, true)
	if len(shorts) != 2 {
		t.Fatalf("expected 2 shorthands, got %v", shorts)
	}
	if shorts[0] != "a" || shorts[1] != "q" {
		t.Fatalf("unexpected shorthands: %v", shorts)
	}
}
