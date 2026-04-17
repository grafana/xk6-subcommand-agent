package agents_test

import (
	"strings"
	"testing"

	"github.com/grafana/xk6-agent/agents"
)

func TestFrontmatter_Render_Minimal(t *testing.T) {
	t.Parallel()

	fm := agents.Frontmatter{
		Name:        "my-agent",
		Description: "A one-line description.",
		Model:       "inherit",
	}
	got := string(fm.Render())

	want := strings.Join([]string{
		"---",
		"name: my-agent",
		"description: A one-line description.",
		"model: inherit",
		"tools: []",
		"---",
		"",
	}, "\n")
	if got != want {
		t.Errorf("Frontmatter.Render() mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFrontmatter_Render_MultilineDescription(t *testing.T) {
	t.Parallel()

	fm := agents.Frontmatter{
		Name:        "multi",
		Description: "line one\nline two\n\nline four",
		Model:       "inherit",
	}
	got := string(fm.Render())

	// Expect literal block scalar, each line indented by two spaces.
	if !strings.Contains(got, "description: |\n") {
		t.Errorf("expected block scalar for multi-line description, got:\n%s", got)
	}
	if !strings.Contains(got, "  line one\n") || !strings.Contains(got, "  line two\n") {
		t.Errorf("expected indented lines, got:\n%s", got)
	}
	// The literal block scalar must not be followed by the raw unindented string.
	if strings.Contains(got, "description: line one\nline two") {
		t.Errorf("raw newlines leaked into scalar, got:\n%s", got)
	}
}

func TestFrontmatter_Render_ToolsList(t *testing.T) {
	t.Parallel()

	fm := agents.Frontmatter{
		Name:        "with-tools",
		Description: "ok description",
		Model:       "inherit",
		Tools:       []string{"k6/info", "k6/run_script"},
	}
	got := string(fm.Render())

	if !strings.Contains(got, "tools:\n  - k6/info\n  - k6/run_script\n") {
		t.Errorf("tools not rendered as a YAML block sequence, got:\n%s", got)
	}
}

func TestFrontmatter_Render_MCPServers(t *testing.T) {
	t.Parallel()

	fm := agents.Frontmatter{
		Name:        "mcp-enabled",
		Description: "ok description",
		Model:       "inherit",
		Tools:       []string{"k6/info"},
		MCPServers: map[string]agents.MCPServerSpec{
			"k6": {
				Type:    "stdio",
				Command: "mcp-k6",
				Args:    []string{},
				Tools:   []string{"*"},
			},
		},
	}
	got := string(fm.Render())

	wantSubstrings := []string{
		"mcp-servers:",
		"  k6:",
		"    type: stdio",
		"    command: mcp-k6",
		"    args: []",
		"    tools:",
		"      - \"*\"",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q in output:\n%s", s, got)
		}
	}
}

func TestFrontmatter_Render_QuotingEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, wantSubstring string
	}{
		{in: "plain", wantSubstring: "name: plain"},
		{in: "true", wantSubstring: `name: "true"`}, // would parse as bool
		{in: "yes", wantSubstring: `name: "yes"`},
		{in: " leading-space", wantSubstring: `name: " leading-space"`},
		{in: "has:colon", wantSubstring: `name: "has:colon"`},
		{in: "- leading-dash", wantSubstring: `name: "- leading-dash"`},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			fm := agents.Frontmatter{Name: tc.in, Description: "ok ok ok", Model: "m"}
			got := string(fm.Render())
			if !strings.Contains(got, tc.wantSubstring) {
				t.Errorf("for input %q: missing %q in:\n%s", tc.in, tc.wantSubstring, got)
			}
		})
	}
}
