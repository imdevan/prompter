package main

import (
	"os"
	"testing"

	"prompter-cli/internal/testutil"
)

func TestPrompterRuns(t *testing.T) {
	if testing.Short() || os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test; set TEST_INTEGRATION=1 to enable")
	}
	testutil.WithTempXDG(t)
	cmd := newRootCmd()
	output, err := testutil.RunCLI(t, cmd, "--yes", "--target=stdout")
	if err != nil {
		t.Fatalf("prompter failed to run: %v\n%s", err, output)
	}
}
