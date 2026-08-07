package ibmbob_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
	"github.com/grafana/xk6-subcommand-agent/agents/mcp"

	_ "github.com/grafana/xk6-subcommand-agent/agents/adapters/ibm_bob"
)

func TestIBMBob_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("ibm-bob")
	if !ok {
		t.Fatal("ibm-bob target not registered")
	}

	if target.DisplayName() != "IBM Bob" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}
}

func TestIBMBob_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("ibm-bob")
	if !ok {
		t.Fatal("ibm-bob target not registered")
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

	// Skills under .bob/skills/ + mcp config.
	skillCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".bob/skills/") {
			skillCount++
		}
	}

	if skillCount < 5 {
		t.Errorf("expected at least 5 skill files, got %d", skillCount)
	}

	// Verify MCP file.
	found := false
	for _, f := range plan.Files {
		if f.Path == ".bob/mcp.json" {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected .bob/mcp.json in plan")
	}
}
