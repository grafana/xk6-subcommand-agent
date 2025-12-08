package agents

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// K6TestGeneratorDescription is the shared description for the k6 test generator agent.
	// Use this agent when the user says "write a k6 script…", "generate a k6 load test…",
	// or "create performance tests with k6."
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

	// K6McpServer is the name of the k6 MCP server.
	K6McpServer = "k6"
)

// AgentConfig represents platform-agnostic agent configuration.
type AgentConfig struct {
	// Name holds the unique identifier of the agent (lowercase letters and hyphens only).
	Name string

	// Description holds a natural language description of the agent's purpose and capabilities.
	Description string

	// Model holds the model to use for this agent.
	Model string

	// Tools holds the tools that the agent can use.
	Tools []string

	// McpServers holds the MCP servers required by this agent.
	McpServers []string

	// Metadata holds additional platform-specific metadata.
	Metadata map[string]any
}

// NewAgentConfig creates and validates a new agent configuration.
func NewAgentConfig(name, description, model string, tools []string) (*AgentConfig, error) {
	config := &AgentConfig{
		Name:        name,
		Description: description,
		Model:       model,
		Tools:       tools,
		McpServers:  []string{},
		Metadata:    make(map[string]any),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the agent configuration.
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

	// Validate tools are non-empty strings
	for i, tool := range c.Tools {
		if strings.TrimSpace(tool) == "" {
			return fmt.Errorf("tool at index %d is empty", i)
		}
	}

	// Validate MCP servers are non-empty strings
	for i, server := range c.McpServers {
		if strings.TrimSpace(server) == "" {
			return fmt.Errorf("MCP server at index %d is empty", i)
		}
	}

	return nil
}

// isValidKebabCase checks if a string is in valid kebab-case format.
func isValidKebabCase(s string) bool {
	// Must start with a letter, contain only lowercase letters, numbers, and hyphens,
	// and end with a letter or number
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, s)
	return matched
}

// ConfigurationFormatter handles platform-specific configuration serialization.
type ConfigurationFormatter interface {
	// FormatAgentConfig formats agent configuration for the platform.
	FormatAgentConfig(config AgentConfig) ([]byte, error)

	// FormatName returns the format name (e.g., "yaml-frontmatter", "json").
	FormatName() string
}

// AgentDefinition combines configuration with content and metadata.
type AgentDefinition struct {
	// Config is the agent configuration.
	Config AgentConfig

	// Content is the agent prompt/instruction content.
	Content []byte

	// Filename is the output filename for this agent.
	Filename string
}

// AgentSpec defines a complete agent specification for initialization.
type AgentSpec struct {
	Config       AgentConfig
	TemplateName string
	OutputPath   string
}

// agentTemplateData holds the data used to render agent template files.
type agentTemplateData struct {
	ConfigHeader string
}
