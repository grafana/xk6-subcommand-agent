package core_test

import (
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents/core"
	"github.com/grafana/xk6-subcommand-agent/agents/mcp"
)

func TestLoadMCPConfig_Valid(t *testing.T) {
	t.Parallel()

	cfg, err := core.LoadMCPConfig(mcp.ServersYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	k6, ok := cfg.Servers["k6"]
	if !ok {
		t.Fatal("expected k6 server to be defined")
	}

	if k6.Command != "k6" {
		t.Errorf("expected command %q, got %q", "k6", k6.Command)
	}

	if len(k6.Args) != 2 || k6.Args[0] != "x" || k6.Args[1] != "mcp" {
		t.Errorf("expected args [x, mcp], got %v", k6.Args)
	}

	if k6.Transport != "stdio" {
		t.Errorf("expected transport %q, got %q", "stdio", k6.Transport)
	}
}

func TestLoadMCPConfig_InvalidYAML(t *testing.T) {
	t.Parallel()

	_, err := core.LoadMCPConfig([]byte(`not: valid: yaml: [`))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadMCPConfig_Empty(t *testing.T) {
	t.Parallel()

	_, err := core.LoadMCPConfig([]byte(`servers: {}`))
	if err == nil {
		t.Fatal("expected error for empty servers")
	}
}

func TestLoadMCPConfig_MissingCommand(t *testing.T) {
	t.Parallel()

	_, err := core.LoadMCPConfig([]byte(`servers:
  test:
    transport: stdio
`))
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestLoadMCPConfig_MissingTransport(t *testing.T) {
	t.Parallel()

	_, err := core.LoadMCPConfig([]byte(`servers:
  test:
    command: test-cmd
`))
	if err == nil {
		t.Fatal("expected error for missing transport")
	}
}
