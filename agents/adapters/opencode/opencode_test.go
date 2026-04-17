package opencode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-agent/agents"
	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
	"github.com/grafana/xk6-agent/agents/mcp"

	_ "github.com/grafana/xk6-agent/agents/adapters/opencode"
)

func TestOpenCode_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("opencode")
	if !ok {
		t.Fatal("opencode target not registered")
	}

	if target.DisplayName() != "OpenCode" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}

	caps := target.Capabilities()
	if !caps.NativeSkills {
		t.Error("expected NativeSkills to be true")
	}

	if caps.MCPConfigPath != "opencode.json" {
		t.Errorf("unexpected MCPConfigPath: %q", caps.MCPConfigPath)
	}
}

func TestOpenCode_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("opencode")
	if !ok {
		t.Fatal("opencode target not registered")
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

	// Should have: 5 SKILL.md files + opencode.json = at least 6
	if len(plan.Files) < 6 {
		t.Errorf("expected at least 6 planned files, got %d", len(plan.Files))
	}

	// Verify skill files are under .opencode/skills/.
	skillCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".opencode/skills/") {
			skillCount++
		}
	}

	if skillCount < 5 {
		t.Errorf("expected at least 5 skill files, got %d", skillCount)
	}

	// Verify opencode.json.
	found := false
	for _, f := range plan.Files {
		if f.Path == "opencode.json" {
			found = true

			if f.Mode != adapters.MergeJSONByKey {
				t.Errorf("expected MergeJSONByKey mode for opencode.json")
			}

			if !strings.Contains(string(f.Content), "opencode.ai") {
				t.Error("expected opencode.json to contain schema URL")
			}

			break
		}
	}

	if !found {
		t.Error("expected opencode.json in plan")
	}
}
