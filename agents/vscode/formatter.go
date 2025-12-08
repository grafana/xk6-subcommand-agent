package vscode

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/grafana/xk6-agent/agents"
)

// Formatter formats agent configurations as YAML frontmatter for VSCode.
type Formatter struct {
	templateContent string
}

// NewFormatter creates a new VSCode frontmatter formatter.
func NewFormatter(templateContent string) *Formatter {
	return &Formatter{
		templateContent: templateContent,
	}
}

// FormatAgentConfig formats agent configuration as YAML frontmatter.
// VSCode format includes MCP server configuration in the frontmatter.
func (f *Formatter) FormatAgentConfig(config agents.AgentConfig) ([]byte, error) {
	// Transform config into VSCode-specific format
	data := vscodeConfigData{
		Name:        config.Name,
		Description: config.Description,
		Tools:       config.Tools,
		Model:       config.Model,
		McpServers:  formatMcpServers(config.McpServers),
	}

	tmpl, err := template.New("vscode-frontmatter").Parse(f.templateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute frontmatter template: %w", err)
	}

	return buf.Bytes(), nil
}

// FormatName returns the format name.
func (f *Formatter) FormatName() string {
	return "vscode-yaml-frontmatter"
}

// vscodeConfigData holds VSCode-specific template data.
type vscodeConfigData struct {
	Name        string
	Description string
	Tools       []string
	Model       string
	McpServers  map[string]McpServerConfig
}

// McpServerConfig represents an MCP server configuration.
type McpServerConfig struct {
	Type    string
	Command string
	Args    []string
	Tools   []string
}

// formatMcpServers transforms MCP server names into full configurations.
func formatMcpServers(serverNames []string) map[string]McpServerConfig {
	servers := make(map[string]McpServerConfig)

	for _, name := range serverNames {
		if name == "k6" {
			servers[name] = McpServerConfig{
				Type:    "stdio",
				Command: "mcp-k6",
				Args:    []string{},
				Tools:   []string{"*"},
			}
		}
		// Add more MCP server configurations as needed
	}

	return servers
}
