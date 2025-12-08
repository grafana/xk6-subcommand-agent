// Package claude provides Claude Code agent initialization.
//
// This package implements the agents.Initializer interface for Claude Code,
// creating the necessary .claude folder structure, agent configurations,
// and project settings.
package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/xk6-agent/agents"
)

const (
	// Claude folder structure constants.
	claudeFolderName          = ".claude"
	claudeAgentsFolderName    = "agents"
	claudeProjectSettingsFile = "settings.local.json"

	// Default Claude configuration values.
	claudeConfigModel = "inherit"
)

// Code is an agent platform implementation for Claude Code.
type Code struct {
	templateLoader agents.TemplateLoader
	fileSystem     agents.FileSystem
	formatter      agents.ConfigurationFormatter
	renderer       *agents.TemplateRenderer
}

// Ensure Code implements the Initializer interface.
var _ agents.Initializer = &Code{}

// NewCode creates a new Code instance with default dependencies.
func NewCode() *Code {
	loader := agents.NewEmbeddedTemplateLoader()

	// Load the frontmatter template
	frontmatterContent, err := loader.LoadContent(agents.TemplateClaudeFrontmatter)
	if err != nil {
		panic(fmt.Sprintf("failed to load frontmatter template: %v", err))
	}

	return &Code{
		templateLoader: loader,
		fileSystem:     &agents.OSFileSystem{},
		formatter:      NewFrontmatterFormatter(string(frontmatterContent)),
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// NewCodeWithDependencies creates a new Code instance with custom dependencies.
// This is primarily useful for testing.
func NewCodeWithDependencies(
	loader agents.TemplateLoader,
	fs agents.FileSystem,
	formatter agents.ConfigurationFormatter,
) *Code {
	return &Code{
		templateLoader: loader,
		fileSystem:     fs,
		formatter:      formatter,
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// Name returns the platform name.
func (c *Code) Name() string {
	return "claude"
}

// Validate checks if initialization can proceed.
func (c *Code) Validate(ctx context.Context, opts agents.InitializerOptions) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	// Ensure we can get the working directory
	workingDir := opts.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = c.fileSystem.WorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	paths := newClaudePaths(workingDir)

	// Check if .claude folder exists and we need Force flag
	if _, err := c.fileSystem.Stat(paths.ClaudeDir()); err == nil && !opts.Force {
		return fmt.Errorf("%s folder already exists at %s, use --force to overwrite", claudeFolderName, paths.ClaudeDir())
	}

	return nil
}

// Initialize initializes the Code agent by creating the .claude folder
// and its contents.
func (c *Code) Initialize(ctx context.Context, opts agents.InitializerOptions) (*agents.InitializeResult, error) {
	result := &agents.InitializeResult{
		FilesCreated: make([]string, 0),
		FilesUpdated: make([]string, 0),
		Warnings:     make([]string, 0),
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("initialization cancelled: %w", err)
	}

	// Get working directory and paths
	paths, err := c.getWorkingPaths(opts)
	if err != nil {
		return nil, err
	}

	// Validate and prepare for initialization
	if err := c.validateAndPrepare(paths, opts, result); err != nil {
		return nil, err
	}

	// If dry-run, stop here
	if opts.DryRun {
		result.Warnings = append(result.Warnings, "Dry-run mode: no files were created")
		return result, nil
	}

	// Create folder structure
	if err := c.createFolderStructure(paths, result); err != nil {
		return nil, err
	}

	// Create settings file
	if err := c.createSettingsFile(paths, result); err != nil {
		return nil, err
	}

	// Initialize agents
	specs := c.getAgentSpecs(paths)
	if err := agents.InitializeAgentSpecs(ctx, c.formatter, c.renderer, c.fileSystem, specs, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Code) getWorkingPaths(opts agents.InitializerOptions) (*claudePaths, error) {
	workingDir := opts.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = c.fileSystem.WorkingDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
	}
	return newClaudePaths(workingDir), nil
}

func (c *Code) validateAndPrepare(
	paths *claudePaths,
	opts agents.InitializerOptions,
	result *agents.InitializeResult,
) error {
	// Check if folder exists and handle based on Force flag
	if _, err := c.fileSystem.Stat(paths.ClaudeDir()); err == nil {
		if !opts.Force {
			return fmt.Errorf(
				"%s folder already exists at %s, use --force to overwrite",
				claudeFolderName,
				paths.ClaudeDir(),
			)
		}
		result.Warnings = append(
			result.Warnings,
			fmt.Sprintf("Removing existing %s folder at %s", claudeFolderName, paths.ClaudeDir()),
		)
	}
	return nil
}

func (c *Code) createFolderStructure(paths *claudePaths, result *agents.InitializeResult) error {
	// Remove existing .claude folder if it exists
	if err := c.fileSystem.RemoveAll(paths.ClaudeDir()); err != nil {
		return fmt.Errorf(
			"failed to remove existing %s folder at %s: %w (check permissions)",
			claudeFolderName,
			paths.ClaudeDir(),
			err,
		)
	}

	// Create .claude folder
	if err := c.fileSystem.Mkdir(paths.ClaudeDir(), 0o755); err != nil {
		return fmt.Errorf(
			"failed to create %s folder at %s: %w (check parent directory exists and is writable)",
			claudeFolderName,
			paths.ClaudeDir(),
			err,
		)
	}
	result.FilesCreated = append(result.FilesCreated, paths.ClaudeDir())

	// Create agents subfolder
	if err := c.fileSystem.Mkdir(paths.AgentsDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create %s folder at %s: %w", claudeAgentsFolderName, paths.AgentsDir(), err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.AgentsDir())

	return nil
}

func (c *Code) getAgentSpecs(paths *claudePaths) []agents.AgentSpec {
	return []agents.AgentSpec{
		{
			Config: agents.AgentConfig{
				Name:        "k6-test-generator",
				Description: agents.K6TestGeneratorDescription,
				Model:       claudeConfigModel,
				Tools:       []string{},
				McpServers:  []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6TestGenerator,
			OutputPath:   paths.AgentFile("k6-test-generator.md"),
		},
		{
			Config: agents.AgentConfig{
				Name:        "k6-playwright-test-converter",
				Description: agents.K6PlaywrightTestConverterDescription,
				Model:       claudeConfigModel,
				Tools:       []string{},
				McpServers:  []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6PlaywrightTestConverter,
			OutputPath:   paths.AgentFile("k6-playwright-test-converter.md"),
		},
	}
}

// createSettingsFile creates the Claude project settings file.
func (c *Code) createSettingsFile(paths *claudePaths, result *agents.InitializeResult) error {
	settings := newClaudeProjectSettings()

	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal claude project settings: %w", err)
	}

	settingsPath := paths.SettingsFile()
	if err := c.fileSystem.WriteFile(settingsPath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write %s file at %s: %w", claudeProjectSettingsFile, settingsPath, err)
	}

	result.FilesCreated = append(result.FilesCreated, settingsPath)
	return nil
}

// claudeProjectSettings holds the settings for the Claude project.
type claudeProjectSettings struct {
	EnabledMcpJSONServers      []string `json:"enabledMcpJsonServers,omitempty"`
	EnableAllProjectMcpServers bool     `json:"enableAllProjectMcpServers"`
}

// newClaudeProjectSettings creates a new claudeProjectSettings instance with default settings.
func newClaudeProjectSettings() *claudeProjectSettings {
	return &claudeProjectSettings{
		EnabledMcpJSONServers:      []string{agents.K6McpServer},
		EnableAllProjectMcpServers: true,
	}
}
