package agents_test

import (
	"testing"

	"github.com/grafana/xk6-agent/agents"
)

func TestAgentConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  agents.AgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
				Tools:       []string{"search", "validate"},
			},
			wantErr: false,
		},
		{
			name: "valid config with single letter name",
			config: agents.AgentConfig{
				Name:        "a",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "invalid - empty name",
			config: agents.AgentConfig{
				Name:        "",
				Description: "Test description",
				Model:       "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid - uppercase in name",
			config: agents.AgentConfig{
				Name:        "Test-Agent",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid - spaces in name",
			config: agents.AgentConfig{
				Name:        "test agent",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid - empty description",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "",
				Model:       "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid - short description",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "Short",
				Model:       "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid - empty model",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "Test description that is long enough",
				Model:       "",
			},
			wantErr: true,
		},
		{
			name: "invalid - empty tool",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
				Tools:       []string{"search", "", "validate"},
			},
			wantErr: true,
		},
		{
			name: "invalid - empty MCP server",
			config: agents.AgentConfig{
				Name:        "test-agent",
				Description: "Test description that is long enough",
				Model:       "gpt-4",
				McpServers:  []string{"k6", "", "database"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAgentConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		agentName   string
		description string
		model       string
		tools       []string
		wantErr     bool
	}{
		{
			name:        "valid config",
			agentName:   "test-agent",
			description: "This is a valid description",
			model:       "gpt-4",
			tools:       []string{"search"},
			wantErr:     false,
		},
		{
			name:        "invalid - bad name",
			agentName:   "Test Agent",
			description: "This is a valid description",
			model:       "gpt-4",
			tools:       []string{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config, err := agents.NewAgentConfig(tt.agentName, tt.description, tt.model, tt.tools)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && config == nil {
				t.Error("NewAgentConfig() returned nil config when expecting valid config")
			}
		})
	}
}
