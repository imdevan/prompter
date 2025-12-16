package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"prompter-cli/internal/config"
)

func main() {
	fmt.Println("Testing Prompter CLI Configuration System")
	fmt.Println("========================================")

	// Create a test config file
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "prompter")
	os.MkdirAll(configDir, 0755)
	
	configPath := filepath.Join(configDir, "test-config.toml")
	testConfig := `
prompts_location = "~/custom/prompts"
editor = "vim"
default_pre = "engineering"
default_post = "testing"
fix_file = "~/fix-output.txt"
max_file_size_bytes = 32768
max_total_bytes = 131072
allow_oversize = true
directory_strategy = "filesystem"
target = "stdout"
`
	
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		log.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(configPath)

	// Test 1: Load config from file
	fmt.Println("\n1. Testing config file loading:")
	manager := config.NewManager()
	cfg, err := manager.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	fmt.Printf("   Editor: %s\n", cfg.Editor)
	fmt.Printf("   Max File Size: %d bytes\n", cfg.MaxFileSizeBytes)
	fmt.Printf("   Directory Strategy: %s\n", cfg.DirectoryStrategy)
	fmt.Printf("   Target: %s\n", cfg.Target)

	// Test 2: Environment variable precedence
	fmt.Println("\n2. Testing environment variable precedence:")
	os.Setenv("PROMPTER_EDITOR", "emacs")
	os.Setenv("PROMPTER_TARGET", "clipboard")
	defer func() {
		os.Unsetenv("PROMPTER_EDITOR")
		os.Unsetenv("PROMPTER_TARGET")
	}()
	
	manager2 := config.NewManager()
	cfg2, err := manager2.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	fmt.Printf("   Editor (env override): %s\n", cfg2.Editor)
	fmt.Printf("   Target (env override): %s\n", cfg2.Target)
	fmt.Printf("   Max File Size (from config): %d bytes\n", cfg2.MaxFileSizeBytes)

	// Test 3: Flag precedence
	fmt.Println("\n3. Testing flag precedence:")
	manager3 := config.NewManager()
	manager3.Load(configPath)
	manager3.SetFlag("editor", "nano")
	manager3.SetFlag("max_file_size_bytes", int64(16384))
	
	cfg3, err := manager3.Resolve()
	if err != nil {
		log.Fatalf("Failed to resolve config: %v", err)
	}
	
	fmt.Printf("   Editor (flag override): %s\n", cfg3.Editor)
	fmt.Printf("   Max File Size (flag override): %d bytes\n", cfg3.MaxFileSizeBytes)
	fmt.Printf("   Target (from env): %s\n", cfg3.Target)

	// Test 4: Validation
	fmt.Println("\n4. Testing validation:")
	err = manager3.Validate(cfg3)
	if err != nil {
		fmt.Printf("   Validation failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Configuration is valid\n")
	}

	// Test 5: Invalid config
	fmt.Println("\n5. Testing invalid configuration:")
	invalidCfg := *cfg3
	invalidCfg.MaxFileSizeBytes = -1
	invalidCfg.DirectoryStrategy = "invalid"
	
	err = manager3.Validate(&invalidCfg)
	if err != nil {
		fmt.Printf("   ✓ Validation correctly caught errors: %v\n", err)
	} else {
		fmt.Printf("   ✗ Validation should have failed\n")
	}

	fmt.Println("\n✓ Configuration system test completed successfully!")
}