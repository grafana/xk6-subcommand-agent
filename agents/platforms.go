// Package agents — see doc.go for the overview.
package agents

import (
	"encoding/json"
	"fmt"
)

// Platforms is the list of targets `k6 x agent` supports. Order matters:
// CLI help and install output follow this sequence.
//
// Adding a new target is a matter of appending another [Platform] literal
// here, mostly declarative except for the per-platform Files function
// that renders frontmatter and config files.
//
//nolint:gochecknoglobals // the descriptor list is the public surface
var Platforms = []Platform{
	claudePlatform(),
	vscodePlatform(),
	opencodePlatform(),
}

// k6MCPServerSpec returns the MCPServerSpec that every platform embeds
// when it needs in-frontmatter MCP config. Keeping this in one place is
// the whole point of the consolidation.
func k6MCPServerSpec() MCPServerSpec {
	return MCPServerSpec{
		Type:    "stdio",
		Command: K6McpCommand,
		Args:    []string{},
		Tools:   []string{"*"},
	}
}

// --- Claude Code ------------------------------------------------------------

func claudePlatform() Platform {
	const (
		claudeModel        = "inherit"
		claudeDir          = ".claude"
		claudeAgentsDir    = ".claude/agents"
		claudeSettingsPath = ".claude/settings.local.json"
	)
	return Platform{
		ID:          "claude",
		DisplayName: "Claude Code",
		Roots:       []string{claudeDir},
		StatusPaths: []StatusPath{
			{Path: claudeAgentsDir, Kind: "non-empty-dir", Description: "agent files"},
		},
		Files: func(shared []AgentConfig) ([]FileSpec, error) {
			out := make([]FileSpec, 0, len(shared)+1)
			// settings.local.json enables the project MCP server(s).
			settings := map[string]any{
				"enabledMcpJsonServers":      []string{K6McpServer},
				"enableAllProjectMcpServers": true,
			}
			content, err := json.MarshalIndent(settings, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal claude settings: %w", err)
			}
			out = append(out, FileSpec{
				Path:    claudeSettingsPath,
				Mode:    0o600,
				Content: content,
			})
			// Per-agent markdown files. Claude ships an inherit-model
			// frontmatter with no tools list (tools are governed by
			// settings.local.json / user grants).
			for _, a := range shared {
				body, err := LoadTemplate(a.BodyTemplate)
				if err != nil {
					return nil, err
				}
				fm := Frontmatter{
					Name:        a.Name,
					Description: a.Description,
					Model:       claudeModel,
				}
				out = append(out, FileSpec{
					Path:    claudeAgentsDir + "/" + a.Name + ".md",
					Mode:    0o644,
					Content: append(fm.Render(), body...),
				})
			}
			return out, nil
		},
	}
}

// --- VS Code / GitHub Copilot -----------------------------------------------

func vscodePlatform() Platform {
	const (
		vscodeModel       = "Claude Sonnet 4"
		githubAgentsDir   = ".github/agents"
		vscodeDir         = ".vscode"
		vscodeMCPFilePath = ".vscode/mcp.json"
	)
	postInstall := `⚠️  To enable the k6 MCP server in GitHub Copilot:
   1. Open github.com › Settings › Copilot › Coding agent
   2. Under "MCP configuration" add the ` + K6McpServer + ` server (same shape as .vscode/mcp.json)
   3. Save your changes`

	return Platform{
		ID:          "vscode",
		DisplayName: "VSCode/GitHub Copilot",
		Roots:       []string{githubAgentsDir, vscodeDir},
		StatusPaths: []StatusPath{
			{Path: githubAgentsDir, Kind: "non-empty-dir", Description: "agent files"},
			{Path: vscodeMCPFilePath, Kind: "file", Description: "mcp config"},
		},
		PostInstall: postInstall,
		Files: func(shared []AgentConfig) ([]FileSpec, error) {
			out := make([]FileSpec, 0, len(shared)+1)

			// .vscode/mcp.json — single source of truth for the MCP
			// server shape that VS Code consumes.
			type mcpServer struct {
				Type    string   `json:"type"`
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}
			mcpDoc := struct {
				Servers map[string]mcpServer `json:"servers"`
				Inputs  []any                `json:"inputs"`
			}{
				Servers: map[string]mcpServer{
					K6McpServer: {Type: "stdio", Command: K6McpCommand, Args: []string{}},
				},
				Inputs: []any{},
			}
			content, err := json.MarshalIndent(mcpDoc, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal vscode mcp.json: %w", err)
			}
			out = append(out, FileSpec{Path: vscodeMCPFilePath, Mode: 0o644, Content: content})

			// Per-agent *.agent.md files. VS Code wants the MCP server
			// in the frontmatter, and tool names are prefixed with the
			// server name.
			mcp := map[string]MCPServerSpec{K6McpServer: k6MCPServerSpec()}
			for _, a := range shared {
				body, err := LoadTemplate(a.BodyTemplate)
				if err != nil {
					return nil, err
				}
				tools := prefixed(K6McpServer+"/", a.Tools)
				fm := Frontmatter{
					Name:        a.Name,
					Description: a.Description,
					Model:       vscodeModel,
					Tools:       tools,
					MCPServers:  mcp,
				}
				out = append(out, FileSpec{
					Path:    githubAgentsDir + "/" + a.Name + ".agent.md",
					Mode:    0o644,
					Content: append(fm.Render(), body...),
				})
			}
			return out, nil
		},
	}
}

// --- OpenCode ---------------------------------------------------------------

func opencodePlatform() Platform {
	const (
		opencodeDir          = ".opencode"
		opencodePromptsDir   = ".opencode/prompts"
		opencodeConfigPath   = "opencode.json"
		opencodeSchemaURL    = "https://opencode.ai/config.json"
		opencodeAgentMode    = "subagent"
		opencodePromptFormat = "{file:.opencode/prompts/%s}"
	)
	return Platform{
		ID:          "opencode",
		DisplayName: "OpenCode",
		// Note: opencode.json lives at the project root and is not
		// inside any Roots directory — see Files for explicit handling.
		Roots: []string{opencodeDir},
		StatusPaths: []StatusPath{
			{Path: opencodePromptsDir, Kind: "non-empty-dir", Description: "prompt files"},
			{Path: opencodeConfigPath, Kind: "file", Description: "opencode config"},
		},
		Files: func(shared []AgentConfig) ([]FileSpec, error) {
			out := make([]FileSpec, 0, len(shared)+1)

			// Per-agent prompt files (plain Markdown, no frontmatter —
			// OpenCode stores agent configuration in opencode.json).
			for _, a := range shared {
				body, err := LoadTemplate(a.BodyTemplate)
				if err != nil {
					return nil, err
				}
				out = append(out, FileSpec{
					Path:    opencodePromptsDir + "/" + a.Name + ".md",
					Mode:    0o644,
					Content: body,
				})
			}

			// opencode.json
			type mcpServer struct {
				Type    string   `json:"type"`
				Command []string `json:"command"`
				Enabled bool     `json:"enabled"`
			}
			type agentEntry struct {
				Description string          `json:"description"`
				Mode        string          `json:"mode"`
				Prompt      string          `json:"prompt"`
				Tools       map[string]bool `json:"tools,omitempty"`
			}
			agentMap := make(map[string]agentEntry, len(shared))
			for _, a := range shared {
				agentMap[a.Name] = agentEntry{
					Description: a.Description,
					Mode:        opencodeAgentMode,
					Prompt:      fmt.Sprintf(opencodePromptFormat, a.Name+".md"),
					Tools:       opencodeToolMap(a.Tools),
				}
			}
			cfg := struct {
				Schema string                `json:"$schema"`
				MCP    map[string]mcpServer  `json:"mcp"`
				Agent  map[string]agentEntry `json:"agent"`
			}{
				Schema: opencodeSchemaURL,
				MCP: map[string]mcpServer{
					K6McpServer: {
						Type:    "local",
						Command: []string{K6McpCommand},
						Enabled: true,
					},
				},
				Agent: agentMap,
			}
			content, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal opencode.json: %w", err)
			}
			out = append(out, FileSpec{Path: opencodeConfigPath, Mode: 0o644, Content: content})
			return out, nil
		},
	}
}

// opencodeToolMap returns the tool-enable map OpenCode expects. It covers
// the built-in editor tools plus one entry per mcp-k6 tool (using the
// server-name-as-prefix glob form OpenCode expects).
func opencodeToolMap(k6ToolNames []string) map[string]bool {
	m := map[string]bool{
		"ls":    true,
		"glob":  true,
		"grep":  true,
		"read":  true,
		"edit":  true,
		"write": true,
	}
	for _, name := range k6ToolNames {
		m[K6McpServer+"*"+name] = true
	}
	return m
}

// prefixed returns a new slice with prefix prepended to each entry.
func prefixed(prefix string, in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = prefix + s
	}
	return out
}
