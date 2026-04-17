package codexcli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/xk6-agent/agents"
	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
	"github.com/grafana/xk6-agent/agents/mcp"

	_ "github.com/grafana/xk6-agent/agents/adapters/codex_cli"
)

func TestCodexCLI_Registered(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("codex-cli")
	if !ok {
		t.Fatal("codex-cli target not registered")
	}

	if target.DisplayName() != "OpenAI Codex CLI" {
		t.Errorf("unexpected display name: %q", target.DisplayName())
	}
}

func TestCodexCLI_Plan(t *testing.T) {
	t.Parallel()

	target, ok := adapters.Get("codex-cli")
	if !ok {
		t.Fatal("codex-cli target not registered")
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

	// Skills under .codex/skills/ + mcp config.
	skillCount := 0
	for _, f := range plan.Files {
		if strings.HasPrefix(f.Path, ".codex/skills/") {
			skillCount++
		}
	}

	if skillCount < 5 {
		t.Errorf("expected at least 5 skill files, got %d", skillCount)
	}

	// Verify MCP file.
	found := false
	for _, f := range plan.Files {
		if f.Path == ".codex/mcp.json" {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected .codex/mcp.json in plan")
	}
}
