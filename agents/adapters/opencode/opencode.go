// Package opencode implements the OpenCode target adapter.
package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters/internal"
)

func init() { adapters.Register(&openCode{}) }

type openCode struct{}

func (openCode) Name() string        { return "opencode" }
func (openCode) DisplayName() string { return "OpenCode" }

func (openCode) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  true,
		MCPConfigPath: "opencode.json",
	}
}

func (o openCode) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. Drop each skill into .opencode/skills/<name>/.
	for _, s := range in.Skills {
		skillFiles, err := internal.PlanSkillFolder(path.Join(".opencode", "skills"), s)
		if err != nil {
			return adapters.Plan{}, fmt.Errorf("opencode: skill %q: %w", s.Name, err)
		}

		files = append(files, skillFiles...)
	}

	// 2. MCP wiring — merge mcp.k6 into opencode.json.
	mcpContent, err := renderOpenCodeConfig(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("opencode: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        "opencode.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "mcp.k6",
		OwnerMarker: "xk6-subcommand-agent:v1",
	})

	return adapters.Plan{Files: files}, nil
}

func renderOpenCodeConfig(in adapters.Inputs) ([]byte, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return nil, fmt.Errorf("k6 MCP server not found in config")
	}

	// OpenCode uses ["command", "arg1", "arg2"] format.
	command := append([]string{k6.Command}, k6.Args...)

	cfg := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"mcp": map[string]any{
			"k6": map[string]any{
				"type":    "local",
				"command": command,
				"enabled": true,
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
