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
	baseDir  string
	registry map[string]string
}

func NewLoader(baseDir string) TemplateRenderer {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		absBase = baseDir
	}

	registry := map[string]string{
		"identity.contract":    "identity/contract.html",
		"identity.certificate": "identity/certificate.html",
		"billing.invoice":      "billing/invoice.html",
	}

	return &Loader{
		baseDir:  absBase,
		registry: registry,
	}
}

func (l *Loader) Render(templateName string, data any) ([]byte, error) {
	relPath, exists := l.registry[templateName]
	if !exists {
		relPath = strings.ReplaceAll(templateName, ".", "/")
		if !strings.HasSuffix(relPath, ".html") {
			relPath += ".html"
		}
	}

	cleanRelPath := filepath.Clean(relPath)
	fullPath := filepath.Join(l.baseDir, cleanRelPath)

	rel, err := filepath.Rel(l.baseDir, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
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
