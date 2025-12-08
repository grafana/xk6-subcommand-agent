package opencode

import (
	"github.com/grafana/xk6-agent/agents"
)

// Formatter formats agent configurations for OpenCode prompts.
// OpenCode stores configuration in opencode.json, so prompt files
// contain only the raw prompt content without frontmatter.
type Formatter struct{}

// NewFormatter creates a new OpenCode formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatAgentConfig returns an empty header since OpenCode stores
// configuration in opencode.json rather than in prompt file frontmatter.
func (f *Formatter) FormatAgentConfig(_ agents.AgentConfig) ([]byte, error) {
	return []byte{}, nil
}

// FormatName returns the format name.
func (f *Formatter) FormatName() string {
	return "opencode-no-frontmatter"
}
