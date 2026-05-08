// Package core provides shared types and helpers for the xk6-subcommand-agent skill
// and adapter system.
package core

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// MCPConfig holds the canonical MCP server definitions loaded from
// agents/mcp/servers.yaml.
type MCPConfig struct {
	Servers map[string]MCPServer `yaml:"servers"`
}

// MCPServer describes a single MCP server that adapters wire into
// target-specific config files.
type MCPServer struct {
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	Env         map[string]string `yaml:"env,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Transport   string            `yaml:"transport"` // "stdio" | "http" | "sse"
}

// LoadMCPConfig parses a servers.yaml file into an MCPConfig.
func LoadMCPConfig(data []byte) (MCPConfig, error) {
	var cfg MCPConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return MCPConfig{}, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return MCPConfig{}, fmt.Errorf("MCP config must define at least one server")
	}

	for name, srv := range cfg.Servers {
		if srv.Command == "" {
			return MCPConfig{}, fmt.Errorf("MCP server %q must have a command", name)
		}

		if srv.Transport == "" {
			return MCPConfig{}, fmt.Errorf("MCP server %q must have a transport", name)
		}
	}

	return cfg, nil
}
