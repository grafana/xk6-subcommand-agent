package cursor_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
	"github.com/grafana/xk6-subcommand-agent/agents/mcp"

	_ "github.com/grafana/xk6-subcommand-agent/agents/adapters/cursor"
)

func TestCursor_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("cursor")
	if !ok {
		t.Fatal("cursor target not registered")
	}

	if target.DisplayName() != "Cursor" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}

	caps := target.Capabilities()
	if caps.NativeSkills {
		t.Error("expected NativeSkills to be false for cursor")
	}
}

func TestCursor_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("cursor")
	if !ok {
		t.Fatal("cursor target not registered")
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
		Root:   "/tmp/test",
	})
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	// Rules under .cursor/rules/ + mcp config.
	ruleCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".cursor/rules/") &&
			strings.HasSuffix(f.Path, ".mdc") {
			ruleCount++
		}
	}

	if ruleCount != 11 {
		t.Errorf("expected 11 cursor rules, got %d", ruleCount)
	}

	// Verify rules have frontmatter wrapper.
	for _, f := range plan.Files {
		if strings.HasSuffix(f.Path, ".mdc") {
			content := string(f.Content)
			if !strings.HasPrefix(content, "---\n") {
				t.Errorf("rule %s missing frontmatter", f.Path)
			}

			if !strings.Contains(content, "alwaysApply:") {
				t.Errorf("rule %s missing alwaysApply", f.Path)
			}
		}
	}

	// Verify MCP file.
	found := false
	for _, f := range plan.Files {
		if f.Path == ".cursor/mcp.json" {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected .cursor/mcp.json in plan")
	}
}
