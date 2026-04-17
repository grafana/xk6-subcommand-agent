// Package vscodecopilot implements the VS Code / GitHub Copilot target adapter.
package vscodecopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/adapters/internal"
)

func init() { adapters.Register(&vscodeCopilot{}) }

type vscodeCopilot struct{}

func (vscodeCopilot) Name() string        { return "vscode-copilot" }
func (vscodeCopilot) DisplayName() string { return "VSCode/GitHub Copilot" }

func (vscodeCopilot) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  true,
		MCPConfigPath: ".vscode/mcp.json",
	}
}

func (v vscodeCopilot) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. Drop each skill into .github/copilot/skills/<name>/.
	for _, s := range in.Skills {
		skillFiles, err := internal.PlanSkillFolder(path.Join(".github", "copilot", "skills"), s)
		if err != nil {
			return adapters.Plan{}, fmt.Errorf("vscode-copilot: skill %q: %w", s.Name, err)
		}

		files = append(files, skillFiles...)
	}

	// 2. MCP wiring — merge into .vscode/mcp.json.
	mcpContent, err := renderVSCodeMCP(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("vscode-copilot: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".vscode/mcp.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "servers.k6",
		OwnerMarker: "xk6-agent:v1",
	})

	return adapters.Plan{Files: files}, nil
}

func renderVSCodeMCP(in adapters.Inputs) ([]byte, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return nil, fmt.Errorf("k6 MCP server not found in config")
	}

	cfg := map[string]any{
		"servers": map[string]any{
			"k6": map[string]any{
				"type":    k6.Transport,
				"command": k6.Command,
				"args":    k6.Args,
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
