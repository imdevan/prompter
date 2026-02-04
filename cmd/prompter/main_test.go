package main

import (
	"testing"

	"prompter-cli/internal/testutil"
)

func TestPrompterRuns(t *testing.T) {
	testutil.WithTempXDG(t)
	cmd := newRootCmd()
	output, err := testutil.RunCLI(t, cmd, "--yes", "--target=stdout")
	if err != nil {
		t.Fatalf("prompter failed to run: %v\n%s", err, output)
	}
}
