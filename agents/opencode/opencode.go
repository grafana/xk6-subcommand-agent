package opencode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/xk6-agent/agents"
)

const (
	openCodeFolderName         = ".opencode"
	openCodePromptsFolderName  = "prompts"
	openCodeConfigFileName     = "opencode.json"
	openCodeSchemaURL          = "https://opencode.ai/config.json"
	openCodeDefaultModel       = "inherit"
	openCodeAgentModeSubagent  = "subagent"
	openCodePromptGenerator    = "k6-test-generator"
	openCodePromptConverter    = "k6-playwright-test-converter"
	openCodePromptGeneratorMD  = openCodePromptGenerator + ".md"
	openCodePromptConverterMD  = openCodePromptConverter + ".md"
	openCodePromptPrefixFormat = "{file:.opencode/prompts/%s}"
)

// OpenCode implements the agents.Initializer interface for OpenCode.
type OpenCode struct {
	templateLoader agents.TemplateLoader
	fileSystem     agents.FileSystem
	formatter      agents.ConfigurationFormatter
	renderer       *agents.TemplateRenderer
}

// Ensure OpenCode implements the Initializer interface.
var _ agents.Initializer = &OpenCode{}

// NewOpenCode creates a new OpenCode instance with default dependencies.
func NewOpenCode() *OpenCode {
	loader := agents.NewEmbeddedTemplateLoader()

	return &OpenCode{
		templateLoader: loader,
		fileSystem:     &agents.OSFileSystem{},
		formatter:      NewFormatter(),
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// NewOpenCodeWithDependencies creates a new OpenCode instance with custom dependencies.
func NewOpenCodeWithDependencies(
	loader agents.TemplateLoader,
	fs agents.FileSystem,
	formatter agents.ConfigurationFormatter,
) *OpenCode {
	return &OpenCode{
		templateLoader: loader,
		fileSystem:     fs,
		formatter:      formatter,
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// Name returns the platform name.
func (o *OpenCode) Name() string {
	return "opencode"
}

// Validate checks if initialization can proceed.
func (o *OpenCode) Validate(ctx context.Context, opts agents.InitializerOptions) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	paths, err := o.getWorkingPaths(opts)
	if err != nil {
		return err
	}

	if _, err := o.fileSystem.Stat(paths.OpenCodeDir()); err == nil && !opts.Force {
		return fmt.Errorf("%s folder already exists at %s, use --force to overwrite", openCodeFolderName, paths.OpenCodeDir())
	}

	return nil
}

// Initialize initializes the OpenCode configuration.
func (o *OpenCode) Initialize(ctx context.Context, opts agents.InitializerOptions) (*agents.InitializeResult, error) {
	result := &agents.InitializeResult{
		FilesCreated: make([]string, 0),
		FilesUpdated: make([]string, 0),
		Warnings:     make([]string, 0),
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("initialization cancelled: %w", err)
	}

	paths, err := o.getWorkingPaths(opts)
	if err != nil {
		return nil, err
	}

	if err := o.prepareForInitialization(paths, opts, result); err != nil {
		return nil, err
	}

	if opts.DryRun {
		result.Warnings = append(result.Warnings, "Dry-run mode: no files were created")
		return result, nil
	}

	if err := o.createFolderStructure(paths, result); err != nil {
		return nil, err
	}

	if err := o.createOpenCodeConfig(paths, result); err != nil {
		return nil, err
	}

	specs := o.getAgentSpecs(paths)
	if err := agents.InitializeAgentSpecs(ctx, o.formatter, o.renderer, o.fileSystem, specs, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (o *OpenCode) getWorkingPaths(opts agents.InitializerOptions) (*openCodePaths, error) {
	workingDir := opts.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = o.fileSystem.WorkingDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
	}
	return newOpenCodePaths(workingDir), nil
}

func (o *OpenCode) prepareForInitialization(paths *openCodePaths, opts agents.InitializerOptions, result *agents.InitializeResult) error {
	if _, err := o.fileSystem.Stat(paths.OpenCodeDir()); err == nil {
		if !opts.Force {
			return fmt.Errorf("%s folder already exists at %s, use --force to overwrite", openCodeFolderName, paths.OpenCodeDir())
		}
		result.Warnings = append(
			result.Warnings,
			fmt.Sprintf("Removing existing %s folder at %s", openCodeFolderName, paths.OpenCodeDir()),
		)
		if err := o.fileSystem.RemoveAll(paths.OpenCodeDir()); err != nil {
			return fmt.Errorf("failed to remove existing %s folder at %s: %w", openCodeFolderName, paths.OpenCodeDir(), err)
		}
	}
	return nil
}

func (o *OpenCode) createFolderStructure(paths *openCodePaths, result *agents.InitializeResult) error {
	if err := o.fileSystem.Mkdir(paths.OpenCodeDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create %s folder at %s: %w", openCodeFolderName, paths.OpenCodeDir(), err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.OpenCodeDir())

	if err := o.fileSystem.Mkdir(paths.PromptsDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create prompts folder at %s: %w", paths.PromptsDir(), err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.PromptsDir())

	return nil
}

func (o *OpenCode) createOpenCodeConfig(paths *openCodePaths, result *agents.InitializeResult) error {
	config := newOpenCodeConfig(paths)

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OpenCode configuration: %w", err)
	}

	if err := o.fileSystem.WriteFile(paths.ConfigFile(), content, 0o644); err != nil {
		return fmt.Errorf("failed to write OpenCode configuration at %s: %w", paths.ConfigFile(), err)
	}

	result.FilesCreated = append(result.FilesCreated, paths.ConfigFile())
	return nil
}

func (o *OpenCode) getAgentSpecs(paths *openCodePaths) []agents.AgentSpec {
	return []agents.AgentSpec{
		{
			Config: agents.AgentConfig{
				Name:        openCodePromptGenerator,
				Description: agents.K6TestGeneratorDescription,
				Model:       openCodeDefaultModel,
				Tools: []string{
					"k6/info",
					"k6/search_documentation",
					"k6/validate_script",
					"k6/run_script",
				},
				McpServers: []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6TestGenerator,
			OutputPath:   paths.PromptFile(openCodePromptGeneratorMD),
		},
		{
			Config: agents.AgentConfig{
				Name:        openCodePromptConverter,
				Description: agents.K6PlaywrightTestConverterDescription,
				Model:       openCodeDefaultModel,
				Tools: []string{
					"k6/info",
					"k6/search_documentation",
					"k6/validate_script",
					"k6/run_script",
				},
				McpServers: []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6PlaywrightTestConverter,
			OutputPath:   paths.PromptFile(openCodePromptConverterMD),
		},
	}
}

// openCodeConfig models the opencode.json structure we emit.
type openCodeConfig struct {
	Schema string                           `json:"$schema"`
	Mcp    map[string]openCodeMcpServer     `json:"mcp"`
	Agent  map[string]openCodeAgentSettings `json:"agent"`
}

type openCodeMcpServer struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
}

type openCodeAgentSettings struct {
	Description string          `json:"description"`
	Mode        string          `json:"mode"`
	Prompt      string          `json:"prompt"`
	Tools       map[string]bool `json:"tools,omitempty"`
}

func newOpenCodeConfig(paths *openCodePaths) openCodeConfig {
	return openCodeConfig{
		Schema: openCodeSchemaURL,
		Mcp: map[string]openCodeMcpServer{
			agents.K6McpServer: {
				Type:    "local",
				Command: []string{"mcp-k6"},
				Enabled: true,
			},
		},
		Agent: map[string]openCodeAgentSettings{
			openCodePromptGenerator: {
				Description: agents.K6TestGeneratorDescription,
				Mode:        openCodeAgentModeSubagent,
				Prompt:      fmt.Sprintf(openCodePromptPrefixFormat, openCodePromptGeneratorMD),
				Tools:       opencodeAgentTools(),
			},
			openCodePromptConverter: {
				Description: agents.K6PlaywrightTestConverterDescription,
				Mode:        openCodeAgentModeSubagent,
				Prompt:      fmt.Sprintf(openCodePromptPrefixFormat, openCodePromptConverterMD),
				Tools:       opencodeAgentTools(),
			},
		},
	}
}

func opencodeAgentTools() map[string]bool {
	return map[string]bool{
		"ls":                      true,
		"glob":                    true,
		"grep":                    true,
		"read":                    true,
		"edit":                    true,
		"write":                   true,
		"k6*info":                 true,
		"k6*search_documentation": true,
		"k6*validate_script":      true,
		"k6*run_script":           true,
	}
}
