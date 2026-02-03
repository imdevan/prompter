package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"prompter-cli/internal/domain"
)

type clipboardStub struct {
	content string
}

func (c *clipboardStub) WriteText(text string) error {
	c.content = text
	return nil
}

type editorStub struct {
	path string
}

func (e *editorStub) Open(path string) error {
	e.path = path
	return nil
}

func TestHandlerWriteStdout(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, nil, nil)
	cfg := domain.DefaultConfig()
	cfg.DisableHistory = true

	if err := handler.Write(domain.Request{Target: "stdout"}, "hello", cfg); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if buf.String() != "hello" {
		t.Fatalf("expected stdout to contain content, got %q", buf.String())
	}
}

func TestHandlerWriteFile(t *testing.T) {
	root := t.TempDir()
	handler := NewHandler(&bytes.Buffer{}, nil, nil)
	cfg := domain.DefaultConfig()
	cfg.DisableHistory = true

	target := "file:" + filepath.Join(root, "output.md")
	if err := handler.Write(domain.Request{Target: target}, "file content", cfg); err != nil {
		t.Fatalf("write file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "output.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "file content" {
		t.Fatalf("expected file content, got %q", string(data))
	}
}

func TestHandlerWriteClipboard(t *testing.T) {
	clipboard := &clipboardStub{}
	handler := NewHandler(&bytes.Buffer{}, clipboard, nil)
	cfg := domain.DefaultConfig()
	cfg.DisableHistory = true

	if err := handler.Write(domain.Request{Target: "clipboard"}, "clip", cfg); err != nil {
		t.Fatalf("write clipboard: %v", err)
	}
	if clipboard.content != "clip" {
		t.Fatalf("expected clipboard content, got %q", clipboard.content)
	}
}

func TestHandlerWriteEditor(t *testing.T) {
	root := t.TempDir()
	editor := &editorStub{}
	handler := NewHandler(&bytes.Buffer{}, nil, editor)
	cfg := domain.DefaultConfig()
	cfg.HistoryLocation = root
	cfg.DisableHistory = false

	if err := handler.Write(domain.Request{Target: "editor", HistorySuffix: "q-t"}, "editor content", cfg); err != nil {
		t.Fatalf("write editor: %v", err)
	}
	if editor.path == "" {
		t.Fatalf("expected editor path to be set")
	}
	data, err := os.ReadFile(editor.path)
	if err != nil {
		t.Fatalf("read editor file: %v", err)
	}
	if string(data) != "editor content" {
		t.Fatalf("expected editor file content, got %q", string(data))
	}
}
