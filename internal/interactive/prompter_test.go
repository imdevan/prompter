package interactive

import (
	"testing"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/testutil"
)

func TestPrompterCollectPreselectsTemplates(t *testing.T) {
	selected := []domain.Template{
		{Name: "question"},
		{Name: "test"},
	}
	ui := testutil.FakeUI{
		BasePrompt: "base",
		Selected:   selected,
	}
	prompter := New(ui)
	req, err := prompter.Collect("base", selected, []string{"question", "test"}, true, "")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(req.TemplateNames) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(req.TemplateNames))
	}
	if req.TemplateNames[0] != "question" || req.TemplateNames[1] != "test" {
		t.Fatalf("unexpected template order: %v", req.TemplateNames)
	}
}
