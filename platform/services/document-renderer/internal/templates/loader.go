package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type TemplateRenderer interface {
	Render(templateName string, data any) ([]byte, error)
}

type Loader struct {
	baseDir string
}

func NewLoader(baseDir string) TemplateRenderer {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		absBase = baseDir
	}
	return &Loader{baseDir: absBase}
}

func (l *Loader) Render(templateName string, data any) ([]byte, error) {
	// Zamienia alias 'identity.contract' na 'identity/contract.html'
	relPath := strings.ReplaceAll(templateName, ".", "/")
	if !strings.HasSuffix(relPath, ".html") {
		relPath += ".html"
	}

	cleanRelPath := filepath.Clean(relPath)
	fullPath := filepath.Join(l.baseDir, cleanRelPath)

	// Ochrona przed Path Traversal
	if !strings.HasPrefix(fullPath, l.baseDir) {
		return nil, fmt.Errorf("invalid template path: access denied")
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	tmpl, err := template.New(filepath.Base(fullPath)).Funcs(funcMap).ParseFiles(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.Bytes(), nil
}
