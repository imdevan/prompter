package template

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// Render applies the given data to a template string.
func Render(content string, data any) (string, error) {
	tmpl, err := template.New("prompt").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			"mdFence": mdFence,
		}).
		Parse(content)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func mdFence(lang string, code any) string {
	return fmt.Sprintf("```%s\n%v\n```", lang, code)
}
