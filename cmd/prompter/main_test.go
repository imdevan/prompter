package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestPrompterRuns(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	cmd := exec.Command("go", "run", ".", "--yes", "--target=stdout")
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+cacheDir, "NO_COLOR=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prompter failed to run: %v\n%s", err, string(output))
	}
}
