// Package cursor implements the Cursor target adapter.
package cursor

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
)

func init() { adapters.Register(&cursorTarget{}) }

type cursorTarget struct{}

func (cursorTarget) Name() string        { return "cursor" }
func (cursorTarget) DisplayName() string { return "Cursor" }

func (cursorTarget) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{
		NativeSkills:  false,
		MCPConfigPath: ".cursor/mcp.json",
	}
}

func (c cursorTarget) Plan(_ context.Context, in adapters.Inputs) (adapters.Plan, error) {
	var files []adapters.PlannedFile

	// 1. For each skill, render a .cursor/rules/<name>.mdc with
	//    YAML frontmatter wrapper + SKILL.md body verbatim.
	for _, s := range in.Skills {
		content := renderCursorRule(s)
		files = append(files, adapters.PlannedFile{
			Path:        path.Join(".cursor", "rules", s.Name+".mdc"),
			Content:     content,
			Mode:        adapters.CreateOnly,
			OwnerMarker: "xk6-agent:v1",
		})
	}

	// 2. MCP wiring — merge into .cursor/mcp.json.
	mcpContent, err := renderCursorMCP(in)
	if err != nil {
		return adapters.Plan{}, fmt.Errorf("cursor: %w", err)
	}

	files = append(files, adapters.PlannedFile{
		Path:        ".cursor/mcp.json",
		Content:     mcpContent,
		Mode:        adapters.MergeJSONByKey,
		MergeKey:    "mcpServers.k6",
		OwnerMarker: "xk6-agent:v1",
	})

	return adapters.Plan{Files: files}, nil
}

func renderCursorRule(s core.Skill) []byte {
	var sb strings.Builder

	// YAML frontmatter wrapper.
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "description: %s\n", s.Description)

	if s.Overrides != nil && s.Overrides.Cursor != nil {
		if len(s.Overrides.Cursor.Globs) > 0 {
			sb.WriteString("globs:\n")
			for _, g := range s.Overrides.Cursor.Globs {
				fmt.Fprintf(&sb, "  - %s\n", g)
			}
		}

		if s.Overrides.Cursor.AlwaysApply {
			sb.WriteString("alwaysApply: true\n")
		} else {
			sb.WriteString("alwaysApply: false\n")
		}
	} else {
		sb.WriteString("alwaysApply: false\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString(s.Body)

	return []byte(sb.String())
}

func renderCursorMCP(in adapters.Inputs) ([]byte, error) {
	k6, ok := in.MCP.Servers["k6"]
	if !ok {
		return nil, fmt.Errorf("k6 MCP server not found in config")
	}

	cfg := map[string]any{
		"mcpServers": map[string]any{
			"k6": map[string]any{
				"command": k6.Command,
				"args":    k6.Args,
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
