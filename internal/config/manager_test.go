package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/utils"
)

func TestManagerLoadPrecedence(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	fixturesRoot := filepath.Join("..", "..", "tests", "config")
	globalSource := filepath.Join(fixturesRoot, "global.toml")
	localSource := filepath.Join(fixturesRoot, "local.toml")

	globalTarget := utils.ConfigPathGlobal()
	localTarget := utils.ConfigPathLocal(cwd)

	if err := copyFile(globalSource, globalTarget); err != nil {
		t.Fatalf("copy global config: %v", err)
	}
	if err := copyFile(localSource, localTarget); err != nil {
		t.Fatalf("copy local config: %v", err)
	}

	manager := NewManager(cwd)
	got, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.PromptsLocation != "/tmp/prompter/global/prompts" {
		t.Fatalf("expected prompts_location from global config, got %q", got.PromptsLocation)
	}
	if got.LocalPromptsLocation != "local-prompts" {
		t.Fatalf("expected local_prompts_location from local config, got %q", got.LocalPromptsLocation)
	}
	if got.Target != "stdout" {
		t.Fatalf("expected target from local config, got %q", got.Target)
	}
	if got.Editor != "nvim" {
		t.Fatalf("expected editor from local config, got %q", got.Editor)
	}
	if got.IncludeAgents != "all" {
		t.Fatalf("expected include_agents from local config, got %q", got.IncludeAgents)
	}
	if got.HistoryClearCycle != "30" {
		t.Fatalf("expected history_clear_cycle from global config, got %q", got.HistoryClearCycle)
	}
	if got.IncludeBuiltinShorthand {
		t.Fatalf("expected include_builtin_shorthand false from global config")
	}
	if got.HistoryLocation != filepath.Join(os.Getenv("XDG_CACHE_HOME"), "prompter", "history") {
		t.Fatalf("expected history_location from defaults, got %q", got.HistoryLocation)
	}
	if got.DirectoryStrategy != "git" {
		t.Fatalf("expected directory_strategy default, got %q", got.DirectoryStrategy)
	}
	if got.Target == "" || got.PromptsLocation == "" {
		t.Fatalf("expected config to be populated")
	}
	if got.RemapShortFlags["directory"] != "D" {
		t.Fatalf("expected remap_short_flags override from local config, got %q", got.RemapShortFlags["directory"])
	}
}

func TestManagerSave(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	manager := NewManager(cwd)
	want := domain.DefaultConfig()
	want.Editor = "vim"
	want.Target = "stdout"
	want.RemapShortFlags = map[string]string{"directory": "d"}

	if err := manager.Save(want); err != nil {
		t.Fatalf("save config: %v", err)
	}

	data, err := os.ReadFile(utils.ConfigPathGlobal())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var got domain.Config
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("saved config mismatch\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestManagerExists(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	manager := NewManager(cwd)
	exists, err := manager.Exists()
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists {
		t.Fatalf("expected no config files to exist yet")
	}

	fixturesRoot := filepath.Join("..", "..", "tests", "config")
	if err := copyFile(filepath.Join(fixturesRoot, "local.toml"), utils.ConfigPathLocal(cwd)); err != nil {
		t.Fatalf("copy local config: %v", err)
	}

	exists, err = manager.Exists()
	if err != nil {
		t.Fatalf("exists after local: %v", err)
	}
	if !exists {
		t.Fatalf("expected local config to be detected")
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
