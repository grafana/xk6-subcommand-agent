//nolint:forbidigo,gosec
package agents

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var embeddedTemplates embed.FS

const (
	// TemplateK6TestGenerator is the path to the k6 test generator template.
	TemplateK6TestGenerator = "templates/k6-test-generator.tmpl.md"
	// TemplateK6PlaywrightTestConverter is the path to the Playwright converter template.
	TemplateK6PlaywrightTestConverter = "templates/k6-playwright-test-converter.tmpl.md"
	// TemplateClaudeFrontmatter is the path to the Claude frontmatter template.
	TemplateClaudeFrontmatter = "templates/claude.configuration.tmpl.frontmatter"
	// TemplateVSCodeFrontmatter is the path to the VSCode frontmatter template.
	TemplateVSCodeFrontmatter = "templates/vscode.configuration.tmpl.frontmatter"
)

// TemplateLoader handles loading templates from various sources.
type TemplateLoader interface {
	// Load loads a template by name and returns a parsed template.
	Load(name string) (*template.Template, error)

	// LoadContent loads raw template content by name.
	LoadContent(name string) ([]byte, error)
}

// FileSystemTemplateLoader loads templates from the file system.
type FileSystemTemplateLoader struct {
	BaseDir string
}

// Load loads a template from the file system.
func (l *FileSystemTemplateLoader) Load(name string) (*template.Template, error) {
	path, err := l.securePath(name)
	if err != nil {
		return nil, err
	}
	return template.ParseFiles(path)
}

// LoadContent loads template content from the file system.
func (l *FileSystemTemplateLoader) LoadContent(name string) ([]byte, error) {
	path, err := l.securePath(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path) // #nosec G304 -- path sanitized by securePath before use
}

func (l *FileSystemTemplateLoader) securePath(name string) (string, error) {
	if l.BaseDir == "" {
		return "", fmt.Errorf("template base directory is not configured")
	}

	if filepath.IsAbs(name) {
		return "", fmt.Errorf("template path %q must be relative", name)
	}

	base, err := filepath.Abs(l.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve template base directory %q: %w", l.BaseDir, err)
	}
	base = filepath.Clean(base)

	cleanName := filepath.Clean(name)
	fullPath := filepath.Join(base, cleanName)

	baseWithSep := base + string(os.PathSeparator)
	if fullPath != base && !strings.HasPrefix(fullPath, baseWithSep) {
		return "", fmt.Errorf("template path %q escapes base directory %q", name, l.BaseDir)
	}

	return fullPath, nil
}

// EmbedTemplateLoader loads templates from an embedded file system.
type EmbedTemplateLoader struct {
	FS embed.FS
}

// NewEmbeddedTemplateLoader returns a TemplateLoader backed by the embedded templates.
func NewEmbeddedTemplateLoader() TemplateLoader {
	return &EmbedTemplateLoader{FS: embeddedTemplates}
}

// Load loads a template from the embedded file system.
func (l *EmbedTemplateLoader) Load(name string) (*template.Template, error) {
	content, err := l.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template %q: %w", name, err)
	}
	return template.New(name).Parse(string(content))
}

// LoadContent loads template content from the embedded file system.
func (l *EmbedTemplateLoader) LoadContent(name string) ([]byte, error) {
	content, err := l.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template %q: %w", name, err)
	}
	return content, nil
}

// TemplateRenderer handles template rendering with arbitrary data.
type TemplateRenderer struct {
	loader TemplateLoader
}

// NewTemplateRenderer creates a new template renderer with the given loader.
func NewTemplateRenderer(loader TemplateLoader) *TemplateRenderer {
	return &TemplateRenderer{loader: loader}
}

// Render renders a template with the given data.
func (r *TemplateRenderer) Render(templateName string, data any) ([]byte, error) {
	tmpl, err := r.loader.Load(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %q: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %q: %w", templateName, err)
	}

	return buf.Bytes(), nil
}

// RenderToFile renders a template to a file with the given permissions.
func (r *TemplateRenderer) RenderToFile(templateName string, data any, destPath string, perm os.FileMode) error {
	content, err := r.Render(templateName, data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, content, perm); err != nil {
		return fmt.Errorf("failed to write file %q: %w", destPath, err)
	}

	return nil
}
