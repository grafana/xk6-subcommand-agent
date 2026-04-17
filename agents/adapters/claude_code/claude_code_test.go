package claudecode_test

import (
	"context"
	"testing"

	"github.com/grafana/xk6-agent/agents"
	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
	"github.com/grafana/xk6-agent/agents/mcp"

	_ "github.com/grafana/xk6-agent/agents/adapters/claude_code"
)

func TestClaudeCode_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("claude-code")
	if !ok {
		t.Fatal("claude-code target not registered")
	}

	if target.DisplayName() != "Claude Code" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}

	caps := target.Capabilities()
	if !caps.NativeSkills {
		t.Error("expected NativeSkills to be true")
	}

	if caps.MCPConfigPath != ".mcp.json" {
		t.Errorf("unexpected MCPConfigPath: %q", caps.MCPConfigPath)
	}
}

func TestClaudeCode_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("claude-code")
	if !ok {
		t.Fatal("claude-code target not registered")
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

	// Should have: 5 SKILL.md files + .mcp.json + settings.local.json = at least 7
	if len(plan.Files) < 7 {
		t.Errorf("expected at least 7 planned files, got %d", len(plan.Files))
	}

	// Verify skill files are under .claude/skills/.
	skillCount := 0
	for _, f := range plan.Files {
		if len(f.Path) > len(".claude/skills/") && f.Path[:len(".claude/skills/")] == ".claude/skills/" {
			skillCount++
		}
	}

	if skillCount < 5 {
		t.Errorf("expected at least 5 skill files under .claude/skills/, got %d", skillCount)
	}

	// Verify MCP and settings files exist with correct modes.
	assertPlannedFile(t, plan, ".mcp.json", adapters.MergeJSONByKey)
	assertPlannedFile(t, plan, ".claude/settings.local.json", adapters.MergeJSONByKey)
}

func assertPlannedFile(t *testing.T, plan adapters.Plan, path string, mode adapters.WriteMode) {
	t.Helper()

	for _, f := range plan.Files {
		if f.Path == path {
			if f.Mode != mode {
				t.Errorf("file %q: expected mode %d, got %d", path, mode, f.Mode)
			}

			if len(f.Content) == 0 {
				t.Errorf("file %q: expected non-empty content", path)
			}

			return
		}
	}

	t.Errorf("expected planned file %q not found", path)
}
