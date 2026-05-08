// Package claudecode implements the Claude Code target adapter.
package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters/internal"
)

func init() { adapters.Register(&claudeCode{}) }

type claudeCode struct{}

func (claudeCode) Name() string        { return "claude-code" }
func (claudeCode) DisplayName() string { return "Claude Code" }

func (claudeCode) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  true,
		MCPConfigPath: ".mcp.json",
	}
}

func (c claudeCode) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. Drop each skill verbatim into .claude/skills/<name>/.
	for _, s := range in.Skills {
		skillFiles, err := internal.PlanSkillFolder(path.Join(".claude", "skills"), s)
		if err != nil {
			return adapters.Plan{}, fmt.Errorf("claude-code: skill %q: %w", s.Name, err)
		}

		files = append(files, skillFiles...)
	}

	// 2. MCP wiring — merge mcpServers.k6 into .mcp.json.
	mcpContent, err := renderMCPJSON(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("claude-code: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".mcp.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "mcpServers.k6",
		OwnerMarker: "xk6-subcommand-agent:v1",
	})

	// 3. settings.local.json — enable our MCP server.
	settingsContent, err := renderSettings()
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("claude-code: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".claude/settings.local.json",
		Content:     settingsContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "enabledMcpJsonServers",
		OwnerMarker: "xk6-subcommand-agent:v1",
	})

	return adapters.Plan{Files: files}, nil
}

func renderMCPJSON(in adapters.Inputs) ([]byte, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return nil, fmt.Errorf("k6 MCP server not found in config")
	}

	mcpEntry := map[string]any{
		"command": k6.Command,
		"args":    k6.Args,
		"type":    k6.Transport,
	}

	cfg := map[string]any{
		"mcpServers": map[string]any{
			"k6": mcpEntry,
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

func renderSettings() ([]byte, error) {
	settings := map[string]any{
		"enabledMcpJsonServers":      []string{"k6"},
		"enableAllProjectMcpServers": true,
	}

	return json.MarshalIndent(settings, "", "  ")
}
