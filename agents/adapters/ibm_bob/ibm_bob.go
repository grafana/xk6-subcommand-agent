// Package ibmbob implements the IBM Bob target adapter.
package ibmbob

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters/internal"
)

func init() { adapters.Register(&ibmBob{}) }

type ibmBob struct{}

func (ibmBob) Name() string        { return "ibm-bob" }
func (ibmBob) DisplayName() string { return "IBM Bob" }

func (ibmBob) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  true,
		MCPConfigPath: ".bob/mcp.json",
	}
}

func (b ibmBob) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. Drop each skill into .bob/skills/<name>/.
	for _, s := range in.Skills {
		skillFiles, err := internal.PlanSkillFolder(path.Join(".bob", "skills"), s)
		if err != nil {
			return adapters.Plan{}, fmt.Errorf("ibm-bob: skill %q: %w", s.Name, err)
		}

		files = append(files, skillFiles...)
	}

	// 2. MCP wiring.
	mcpContent, err := renderBobMCP(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("ibm-bob: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".bob/mcp.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    adapters.MCPServerKey,
		OwnerMarker: adapters.OwnerMarker,
	})

	return adapters.Plan{Files: files}, nil
}

func renderBobMCP(in adapters.Inputs) ([]byte, error) {
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
