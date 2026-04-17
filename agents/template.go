// Package agents — see doc.go for the overview.
package agents

import (
	"embed"
	"fmt"
)

//go:embed templates/*
var embeddedTemplates embed.FS

// LoadTemplate reads a file from the embedded templates filesystem. Names
// are relative to the package root (e.g. "templates/k6-test-generator.md").
func LoadTemplate(name string) ([]byte, error) {
	content, err := embeddedTemplates.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("load embedded template %q: %w", name, err)
	}
	return content, nil
}
