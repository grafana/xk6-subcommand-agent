package vscodecopilot_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
	"github.com/grafana/xk6-subcommand-agent/agents/mcp"

	_ "github.com/grafana/xk6-subcommand-agent/agents/adapters/vscode_copilot"
)

func TestVSCodeCopilot_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("vscode-copilot")
	if !ok {
		t.Fatal("vscode-copilot target not registered")
	}

	if target.DisplayName() != "VSCode/GitHub Copilot" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}

	caps := target.Capabilities()
	if !caps.NativeSkills {
		t.Error("expected NativeSkills to be true")
	}

	if caps.MCPConfigPath != ".vscode/mcp.json" {
		t.Errorf("unexpected MCPConfigPath: %q", caps.MCPConfigPath)
	}
}

func TestVSCodeCopilot_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("vscode-copilot")
	if !ok {
		t.Fatal("vscode-copilot target not registered")
	}

	skills, err := core.LoadSkills(agents.SkillsFS)
	if err != nil {
		t.Fatalf("failed to load skills: %v", err)
	}

	mcpCfg, err := core.LoadMCPConfig(mcp.ServersYAML)
	if err != nil {
		t.Fatalf("failed to load MCP config: %v", err)
	}

	plan, err := target.Plan(context.Background(), adapters.Inputs{
		Skills: skills,
		MCP:    mcpCfg,
		Root:   "/tmp/test-project",
	})
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	// Should have: 5 SKILL.md files + .vscode/mcp.json = at least 6
	if len(plan.Files) < 6 {
		t.Errorf("expected at least 6 planned files, got %d", len(plan.Files))
	}

	// Verify skill files are under .github/copilot/skills/.
	skillCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".github/copilot/skills/") {
			skillCount++
		}
	}

	if skillCount < 5 {
		t.Errorf("expected at least 5 skill files, got %d", skillCount)
	}

	// Verify MCP file.
	found := false
	for _, f := range plan.Files {
		if f.Path == ".vscode/mcp.json" {
			found = true

			if f.Mode != adapters.MergeJSONByKey {
				t.Errorf("expected MergeJSONByKey mode for mcp.json")
			}

			break
		}
	}

	if !found {
		t.Error("expected .vscode/mcp.json in plan")
	}
}
