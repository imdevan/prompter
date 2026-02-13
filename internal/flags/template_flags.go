package flags

import (
	"path/filepath"
	"strings"

	"prompter-cli/internal/domain"
)

// TemplateFlagInfo captures resolved flag metadata for templates.
type TemplateFlagInfo struct {
	Flag      string
	Shorthand string
}

// TemplateFlags resolves flag/shorthand assignments, mutating usedShort to avoid collisions.
func TemplateFlags(cfg domain.Config, templates []domain.Template, usedShort map[string]bool) map[string]TemplateFlagInfo {
	if usedShort == nil {
		usedShort = map[string]bool{}
	}
	info := make(map[string]TemplateFlagInfo, len(templates))
	for _, tmpl := range templates {
		flagName := strings.TrimSpace(tmpl.Flag)
		if flagName == "" {
			flagName = DefaultTemplateFlagName(tmpl.Name)
		}
		if flagName == "" {
			continue
		}
		shorthand := strings.TrimSpace(tmpl.Shorthand)
		if shorthand != "" && len([]rune(shorthand)) != 1 {
			shorthand = ""
		}
		if shorthand != "" && usedShort[shorthand] {
			shorthand = ""
		}
		if shorthand == "" {
			shorthand = TemplateShorthandFromName(tmpl.Name, usedShort)
		}
		if shorthand != "" {
			usedShort[shorthand] = true
		}
		info[tmpl.Name] = TemplateFlagInfo{
			Flag:      flagName,
			Shorthand: shorthand,
		}
	}
	return info
}

// BuiltinShortFlag resolves built-in shorthand with config overrides.
func BuiltinShortFlag(cfg domain.Config, longName, defaultShort string) string {
	if cfg.RemapShortFlags != nil {
		if remapped, ok := cfg.RemapShortFlags[longName]; ok {
			remapped = strings.TrimSpace(remapped)
			if len([]rune(remapped)) == 1 {
				return remapped
			}
		}
	}
	if !cfg.IncludeBuiltinShorthand {
		return ""
	}
	return defaultShort
}

// BuiltinShortFlags returns used builtin shorthand flags for collision checks.
func BuiltinShortFlags(cfg domain.Config) map[string]bool {
	used := map[string]bool{}
	for _, entry := range []struct {
		long     string
		defShort string
	}{
		{"config", ""},
		{"file", ""},
		{"directory", "D"},
		{"target", "T"},
		{"clipboard", "B"},
		{"agents", "A"},
		{"interactive", "I"},
		{"yes", "Y"},
		{"editor", "E"},
		{"template", ""},
		{"tag", "G"},
		{"version", "V"},
	} {
		if shorthand := BuiltinShortFlag(cfg, entry.long, entry.defShort); shorthand != "" {
			used[shorthand] = true
		}
	}
	return used
}

// DefaultTemplateFlagName converts a template name into a flag name.
func DefaultTemplateFlagName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	if trimmed := templateFlagNameBase(name); trimmed != "" {
		return normalizeFlagName(trimmed)
	}
	return normalizeFlagName(name)
}

func templateFlagNameBase(name string) string {
	path := filepath.ToSlash(strings.TrimSpace(name))
	if path == "" {
		return ""
	}
	lower := strings.ToLower(path)
	for _, prefix := range []string{
		"agents/skills/",
		"claude/skills/",
		"cursor/skills/",
		"kiro/skills/",
		"opencode/skills/",
		"antigravity/skills/",
	} {
		if strings.HasPrefix(lower, prefix) {
			suffix := path[len(prefix):]
			return strings.TrimSuffix(suffix, "/")
		}
	}
	for _, prefix := range []string{
		"cursor/commands/",
	} {
		if strings.HasPrefix(lower, prefix) {
			suffix := path[len(prefix):]
			suffix = strings.TrimSuffix(suffix, filepath.Ext(suffix))
			return strings.TrimSuffix(suffix, "/")
		}
	}
	return ""
}

func normalizeFlagName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

// TemplateShorthandFromName derives a shorthand from the template name.
func TemplateShorthandFromName(name string, used map[string]bool) string {
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" {
		return ""
	}
	for _, r := range base {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if r < 'a' || r > 'z' {
			continue
		}
		letter := string(r)
		if !used[letter] {
			return letter
		}
	}
	return ""
}
