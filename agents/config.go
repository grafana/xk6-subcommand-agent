// Package agents — see doc.go for the overview.
package agents

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// K6TestGeneratorDescription is the shared description for the k6 test generator agent.
	K6TestGeneratorDescription = `Use this agent when the user says "write a k6 script…", ` +
		`"generate a k6 load test…", or "create performance tests with k6." ` +
		`Handles load, performance, stress, soak, and reliability tests for APIs, services, ` +
		`and applications, leveraging mcp-k6 for script validation and documentation lookup. Examples:

<example>
Context: User: "Write a k6 script testing quickpizza.grafana.com and ensuring ` +
		`it sustains reasonable load and performs as expected."
assistant: "I'll use the k6-test-generator agent to craft a resilient load test ` +
		`for quickpizza.grafana.com."
</example>

<example>
Context: User: "Generate a k6 load test for our checkout microservice that models peak traffic."
assistant: "I'll delegate to the k6-test-generator agent to design the k6 scenarios ` +
		`and scripts for your checkout flow."
</example>`

	// K6PlaywrightTestConverterDescription is the shared description for the Playwright converter agent.
	K6PlaywrightTestConverterDescription = `Use this agent when a user provides a Playwright script ` +
		`and needs a faithful conversion into a production-ready k6/browser test while ` +
		`following Grafana's migration guidance and MCP workflows.`

	// K6McpServer is the canonical name used for the k6 MCP server across every
	// platform configuration this extension emits.
	K6McpServer = "k6"

	// K6McpCommand is the executable the MCP server runs. It must be on the
	// user's PATH for the generated configurations to work.
	K6McpCommand = "mcp-k6"
)

// K6Tools returns the canonical list of mcp-k6 tool names that every agent
// is granted. Platforms that need a different naming convention (e.g. glob
// syntax) derive their form from this list. Returns a fresh slice so callers
// can mutate it safely.
func K6Tools() []string {
	return []string{
		"info",
		"search_documentation",
		"validate_script",
		"run_script",
	}
}

// AgentConfig describes a single agent shared across platforms.
//
// Name is the kebab-case identifier of the agent. Description is the
// natural-language trigger the host IDE shows to the user. BodyTemplate is
// the path (relative to the embedded templates/ filesystem) of the Markdown
// body that gets combined with per-platform frontmatter at render time.
type AgentConfig struct {
	Name         string
	Description  string
	Model        string
	Tools        []string
	McpServers   []string
	BodyTemplate string
}

// NewAgentConfig constructs a validated AgentConfig. Mainly useful for tests.
func NewAgentConfig(name, description, model string, tools []string) (*AgentConfig, error) {
	cfg := &AgentConfig{
		Name:        name,
		Description: description,
		Model:       model,
		Tools:       tools,
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// Validate checks the agent configuration for obvious issues before rendering.
func (c *AgentConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !isValidKebabCase(c.Name) {
		return fmt.Errorf("name must be lowercase with hyphens only (kebab-case): got %q", c.Name)
	}
	if c.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(c.Description) < 10 {
		return fmt.Errorf("description must be at least 10 characters long: got %d characters", len(c.Description))
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	for i, tool := range c.Tools {
		if strings.TrimSpace(tool) == "" {
			return fmt.Errorf("tool at index %d is empty", i)
		}
	}
	for i, server := range c.McpServers {
		if strings.TrimSpace(server) == "" {
			return fmt.Errorf("MCP server at index %d is empty", i)
		}
	}
	return nil
}

// SharedAgents returns the agent list that every platform scaffolds. It
// returns fresh slices so callers can safely mutate per-platform fields
// (model, tool names) without touching shared state.
func SharedAgents() []AgentConfig {
	return []AgentConfig{
		{
			Name:         "k6-test-generator",
			Description:  K6TestGeneratorDescription,
			Tools:        K6Tools(),
			McpServers:   []string{K6McpServer},
			BodyTemplate: "templates/k6-test-generator.md",
		},
		{
			Name:         "k6-playwright-test-converter",
			Description:  K6PlaywrightTestConverterDescription,
			Tools:        K6Tools(),
			McpServers:   []string{K6McpServer},
			BodyTemplate: "templates/k6-playwright-test-converter.md",
		},
	}
}

// isValidKebabCase enforces lowercase-letters-and-dashes agent names.
func isValidKebabCase(s string) bool {
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, s)
	return matched
}
