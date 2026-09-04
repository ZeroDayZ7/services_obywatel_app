package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
)

type Loader struct {
	baseDir string
}

func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

func (l *Loader) Render(templateRelativePath string, data any) ([]byte, error) {
	fullPath := filepath.Join(l.baseDir, templateRelativePath)

	// Można dodać globalne helpery dla szablonów HTML
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	tmpl, err := template.New(filepath.Base(fullPath)).Funcs(funcMap).ParseFiles(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templateRelativePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateRelativePath, err)
	}

	return buf.Bytes(), nil
}
