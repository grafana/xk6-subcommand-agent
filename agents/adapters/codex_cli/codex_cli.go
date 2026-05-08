// Package codexcli implements the OpenAI Codex CLI target adapter.
package codexcli

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters/internal"
)

func init() { adapters.Register(&codexCLI{}) }

type codexCLI struct{}

func (codexCLI) Name() string        { return "codex-cli" }
func (codexCLI) DisplayName() string { return "OpenAI Codex CLI" }

func (codexCLI) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  true,
		MCPConfigPath: ".codex/mcp.json",
	}
}

func (c codexCLI) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. Drop each skill into .codex/skills/<name>/.
	for _, s := range in.Skills {
		skillFiles, err := internal.PlanSkillFolder(path.Join(".codex", "skills"), s)
		if err != nil {
			return adapters.Plan{}, fmt.Errorf("codex-cli: skill %q: %w", s.Name, err)
		}

		files = append(files, skillFiles...)
	}

	// 2. MCP wiring.
	mcpContent, err := renderCodexMCP(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("codex-cli: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".codex/mcp.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "mcpServers.k6",
		OwnerMarker: "xk6-subcommand-agent:v1",
	})

	return adapters.Plan{Files: files}, nil
}

func renderCodexMCP(in adapters.Inputs) ([]byte, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return nil, fmt.Errorf("k6 MCP server not found in config")
	}

	cfg := map[string]any{
		"mcpServers": map[string]any{
			"k6": map[string]any{
				"command": k6.Command,
				"args":    k6.Args,
				"type":    k6.Transport,
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
