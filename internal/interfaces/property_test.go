package interfaces

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Test that gopter property-based testing framework is properly set up
func TestGopterSetup(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Simple property test to verify gopter is working
	properties.Property("string length is non-negative", prop.ForAll(
		func(s string) bool {
			return len(s) >= 0
		},
		gen.AnyString(),
	))

	// Run with minimum 100 iterations as specified in design
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}