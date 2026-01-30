package main

import (
	"os/exec"
	"testing"
)

func TestPrompterRuns(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "run", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prompter failed to run: %v\n%s", err, string(output))
	}
}
