package claude

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/grafana/xk6-agent/agents"
)

// FrontmatterFormatter formats agent configurations as YAML frontmatter.
type FrontmatterFormatter struct {
	templateContent string
}

// NewFrontmatterFormatter creates a new frontmatter formatter.
func NewFrontmatterFormatter(templateContent string) *FrontmatterFormatter {
	return &FrontmatterFormatter{
		templateContent: templateContent,
	}
}

// FormatAgentConfig formats agent configuration as YAML frontmatter.
func (f *FrontmatterFormatter) FormatAgentConfig(config agents.AgentConfig) ([]byte, error) {
	tmpl, err := template.New("frontmatter").Parse(f.templateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute frontmatter template: %w", err)
	}

	return buf.Bytes(), nil
}

// FormatName returns the format name.
func (f *FrontmatterFormatter) FormatName() string {
	return "yaml-frontmatter"
}
