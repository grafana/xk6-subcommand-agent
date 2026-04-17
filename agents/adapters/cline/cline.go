// Package cline implements the Cline target adapter.
package cline

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
)

func init() { adapters.Register(&clineTarget{}) }

type clineTarget struct{}

func (clineTarget) Name() string        { return "cline" }
func (clineTarget) DisplayName() string { return "Cline" }

func (clineTarget) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills: false,
		// MCP is global for Cline, not project-scoped.
		MCPConfigPath: "",
	}
}

func (c clineTarget) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	files := make([]adapters.PlannedFile, 0, len(in.Skills))
	notices := make([]string, 0, 3)

	// 1. For each skill, write .clinerules/<name>.md with a header
	//    + SKILL.md body verbatim.
	for _, s := range in.Skills {
		content := renderClineRule(s)
		files = append(files, adapters.PlannedFile{
			Path:        path.Join(".clinerules", s.Name+".md"),
			Content:     content,
			Mode:        adapters.CreateOnly,
			OwnerMarker: "xk6-agent:v1",
		})
	}

	// 2. MCP is global for Cline — print the snippet as a notice.
	mcpSnippet, err := renderClineMCPSnippet(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("cline: %w", err)
	}

	notices = append(notices,
		"Cline MCP config is global (not project-scoped).",
		"Add the following to your cline_mcp_settings.json:",
		mcpSnippet,
	)

	return adapters.Plan{Files: files, Notices: notices}, nil
}

func renderClineRule(s core.Skill) []byte {
	var sb strings.Builder

	fmt.Fprintf(&sb, "<!-- Loaded from k6-agent skill: %s -->\n\n", s.Name)
	sb.WriteString(s.Body)

	return []byte(sb.String())
}

func renderClineMCPSnippet(in adapters.Inputs) (string, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return "", fmt.Errorf("k6 MCP server not found in config")
	}

	snippet := map[string]any{
		"k6": map[string]any{
			"command": k6.Command,
			"args":    k6.Args,
		},
	}

	data, err := json.MarshalIndent(snippet, "  ", "  ")
	if err != nil {
		return "", err
	}

	return "  " + string(data), nil
}
