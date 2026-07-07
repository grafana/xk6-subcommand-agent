package cline_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
	"github.com/grafana/xk6-subcommand-agent/agents/mcp"

	_ "github.com/grafana/xk6-subcommand-agent/agents/adapters/cline"
)

func TestCline_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("cline")
	if !ok {
		t.Fatal("cline target not registered")
	}

	if target.DisplayName() != "Cline" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}

	caps := target.Capabilities()
	if caps.MCPConfigPath != "" {
		t.Error("expected empty MCPConfigPath for cline (global MCP)")
	}
}

func TestCline_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("cline")
	if !ok {
		t.Fatal("cline target not registered")
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

	// Rules under .clinerules/.
	ruleCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".clinerules/") {
			ruleCount++

			// Verify header comment.
			content := string(f.Content)
			if !strings.Contains(content, "Loaded from k6-agent skill") {
				t.Errorf("rule %s missing header comment", f.Path)
			}
		}
	}

	if ruleCount != 10 {
		t.Errorf("expected 10 cline rules, got %d", ruleCount)
	}

	// Cline should have notices about global MCP config.
	if len(plan.Notices) == 0 {
		t.Error("expected notices about global MCP config")
	}

	hasGlobalNotice := false
	for _, n := range plan.Notices {
		if strings.Contains(n, "global") {
			hasGlobalNotice = true

			break
		}
	}

	if !hasGlobalNotice {
		t.Error("expected notice mentioning global MCP config")
	}
}
