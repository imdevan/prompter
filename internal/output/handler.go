package output

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"prompter-cli/internal/domain"
	"prompter-cli/internal/template"
)

// Handler routes generated prompts to the configured target.
type Handler struct {
	Stdout    io.Writer
	Clipboard Clipboard
	Editor    Editor
}

// NewHandler builds an output handler.
func NewHandler(stdout io.Writer, clipboard Clipboard, editor Editor) *Handler {
	return &Handler{
		Stdout:    stdout,
		Clipboard: clipboard,
		Editor:    editor,
	}
}

// Write sends content to the requested target, falling back to config.Target.
func (h *Handler) Write(req domain.Request, content string, cfg domain.Config) error {
	target := req.Target
	if target == "" {
		target = cfg.Target
	}

	if !cfg.DisableHistory {
		if strings.TrimSpace(content) != "" {
			if _, err := h.writeHistory(content, cfg, req.HistorySuffix, req.HistoryTag); err != nil {
				return err
			}
		}
	}
	switch {
	case target == "stdout":
		if h.Stdout == nil {
			return errors.New("stdout writer is required")
		}
		_, err := io.Copy(h.Stdout, bytes.NewBufferString(content))
		return err
	case target == "clipboard":
		if req.EditorTarget {
			if h.Editor == nil {
				return errors.New("editor adapter is required")
			}
			if h.Clipboard == nil {
				return errors.New("clipboard adapter is required")
			}
			path, err := h.openInEditor(content, cfg, req.HistorySuffix, req.HistoryTag)
			if err != nil {
				return err
			}
			if !req.EditorIsVim {
				return h.Clipboard.WriteText(content)
			}
			return h.copyBodyAfterFrontmatter(path)
		}
		if h.Clipboard == nil {
			return errors.New("clipboard adapter is required")
		}
		return h.Clipboard.WriteText(content)
	case target == "editor":
		_, err := h.openInEditor(content, cfg, req.HistorySuffix, req.HistoryTag)
		return err
	case strings.HasPrefix(target, "file:"):
		path := strings.TrimPrefix(target, "file:")
		if strings.TrimSpace(path) == "" {
			return errors.New("file target path is required")
		}
		return writeFile(path, content)
	default:
		return fmt.Errorf("unsupported target: %s", target)
	}
}

func (h *Handler) openInEditor(content string, cfg domain.Config, suffix, tag string) (string, error) {
	if h.Editor == nil {
		return "", errors.New("editor adapter is required")
	}
	dir := cfg.HistoryLocation
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("prompt-%s", time.Now().Format("20060102-150405"))
	if strings.TrimSpace(suffix) != "" {
		filename = filename + "-" + strings.TrimSpace(suffix)
	}
	filename += ".md"
	path := filepath.Join(dir, filename)
	if err := writeFile(path, withHistoryFrontmatter(content, tag)); err != nil {
		return "", err
	}
	if opener, ok := h.Editor.(interface{ OpenAtEnd(string) error }); ok {
		return path, opener.OpenAtEnd(path)
	}
	return path, h.Editor.Open(path)
}

func (h *Handler) writeHistory(content string, cfg domain.Config, suffix, tag string) (string, error) {
	dir := cfg.HistoryLocation
	if strings.TrimSpace(dir) == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("prompt-%s", time.Now().Format("20060102-150405"))
	if strings.TrimSpace(suffix) != "" {
		filename = filename + "-" + strings.TrimSpace(suffix)
	}
	filename += ".md"
	path := filepath.Join(dir, filename)
	if err := writeFile(path, withHistoryFrontmatter(content, tag)); err != nil {
		return "", err
	}
	return path, nil
}

func withHistoryFrontmatter(content, tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return content
	}
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("tag: ")
	builder.WriteString(fmt.Sprintf("%q", tag))
	builder.WriteString("\n---\n\n")
	builder.WriteString(content)
	return builder.String()
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (h *Handler) copyBodyAfterFrontmatter(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	body := template.StripFrontmatter(string(data))
	body = strings.TrimSpace(body)
	return h.Clipboard.WriteText(body)
}
