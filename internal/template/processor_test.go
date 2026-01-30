package template

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	content := "Hello {{ .Name }}\n\n{{ mdFence \"go\" .Code }}"
	out, err := Render(content, map[string]any{
		"Name": "Prompter",
		"Code": "fmt.Println(\"hi\")",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "Hello Prompter"; !strings.Contains(out, want) {
		t.Fatalf("expected output to contain %q, got %q", want, out)
	}
	if want := "```go"; !strings.Contains(out, want) {
		t.Fatalf("expected output to contain %q, got %q", want, out)
	}
}
