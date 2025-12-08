// Package vscode provides VSCode/GitHub Copilot agent initialization.
//
// This package implements the agents.Initializer interface for VSCode,
// creating the necessary .github/agents and .vscode folder structure,
// agent configurations, and MCP settings.
package vscode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/xk6-agent/agents"
)

const (
	// VSCode folder structure constants.
	githubFolderName  = ".github"
	agentsFolderName  = "agents"
	vscodeFolderName  = ".vscode"
	mcpConfigFileName = "mcp.json"

	// Default VSCode configuration values.
	vscodeConfigModel = "Claude Sonnet 4"
)

// VSCode is an agent platform implementation for VSCode/GitHub Copilot.
type VSCode struct {
	templateLoader agents.TemplateLoader
	fileSystem     agents.FileSystem
	formatter      agents.ConfigurationFormatter
	renderer       *agents.TemplateRenderer
}

// Ensure VSCode implements the Initializer interface.
var _ agents.Initializer = &VSCode{}

// NewVSCode creates a new VSCode instance with default dependencies.
func NewVSCode() *VSCode {
	loader := agents.NewEmbeddedTemplateLoader()

	// Load the frontmatter template
	frontmatterContent, err := loader.LoadContent(agents.TemplateVSCodeFrontmatter)
	if err != nil {
		panic(fmt.Sprintf("failed to load frontmatter template: %v", err))
	}

	return &VSCode{
		templateLoader: loader,
		fileSystem:     &agents.OSFileSystem{},
		formatter:      NewFormatter(string(frontmatterContent)),
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// NewVSCodeWithDependencies creates a new VSCode instance with custom dependencies.
// This is primarily useful for testing.
func NewVSCodeWithDependencies(
	loader agents.TemplateLoader,
	fs agents.FileSystem,
	formatter agents.ConfigurationFormatter,
) *VSCode {
	return &VSCode{
		templateLoader: loader,
		fileSystem:     fs,
		formatter:      formatter,
		renderer:       agents.NewTemplateRenderer(loader),
	}
}

// Name returns the platform name.
func (v *VSCode) Name() string {
	return "vscode"
}

// Validate checks if initialization can proceed.
func (v *VSCode) Validate(ctx context.Context, opts agents.InitializerOptions) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	// Ensure we can get the working directory
	workingDir := opts.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = v.fileSystem.WorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	paths := newVSCodePaths(workingDir)

	// Check if folders exist and we need Force flag
	agentsDirExists := false
	vscodeDirExists := false

	if _, err := v.fileSystem.Stat(paths.AgentsDir()); err == nil {
		agentsDirExists = true
	}
	if _, err := v.fileSystem.Stat(paths.VSCodeDir()); err == nil {
		vscodeDirExists = true
	}

	if (agentsDirExists || vscodeDirExists) && !opts.Force {
		return fmt.Errorf("VSCode agent folders already exist, use --force to overwrite")
	}

	return nil
}

// Initialize initializes the VSCode agent by creating the necessary folders and files.
func (v *VSCode) Initialize(ctx context.Context, opts agents.InitializerOptions) (*agents.InitializeResult, error) {
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
	paths, err := v.getWorkingPaths(opts)
	if err != nil {
		return nil, err
	}

	// Validate existing folders
	if err := v.validateExistingFolders(paths, opts, result); err != nil {
		return nil, err
	}

	// If dry-run, stop here
	if opts.DryRun {
		result.Warnings = append(result.Warnings, "Dry-run mode: no files were created")
		return result, nil
	}

	// Create folder structures and MCP config
	if err := v.createFolderStructures(paths, result); err != nil {
		return nil, err
	}

	// Initialize agents
	specs := v.getAgentSpecs(paths)
	if err := agents.InitializeAgentSpecs(ctx, v.formatter, v.renderer, v.fileSystem, specs, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (v *VSCode) getWorkingPaths(opts agents.InitializerOptions) (*vscodePaths, error) {
	workingDir := opts.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = v.fileSystem.WorkingDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
	}
	return newVSCodePaths(workingDir), nil
}

func (v *VSCode) createFolderStructures(paths *vscodePaths, result *agents.InitializeResult) error {
	if err := v.createGitHubStructure(paths, result); err != nil {
		return err
	}
	if err := v.createVSCodeStructure(paths, result); err != nil {
		return err
	}
	return v.createMcpConfig(paths, result)
}

func (v *VSCode) getAgentSpecs(paths *vscodePaths) []agents.AgentSpec {
	return []agents.AgentSpec{
		{
			Config: agents.AgentConfig{
				Name:        "k6-test-generator",
				Description: agents.K6TestGeneratorDescription,
				Model:       vscodeConfigModel,
				Tools: []string{
					"k6/info",
					"k6/search_documentation",
					"k6/validate_script",
					"k6/run_script",
				},
				McpServers: []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6TestGenerator,
			OutputPath:   paths.AgentFile("k6-test-generator.agent.md"),
		},
		{
			Config: agents.AgentConfig{
				Name:        "k6-playwright-test-converter",
				Description: agents.K6PlaywrightTestConverterDescription,
				Model:       vscodeConfigModel,
				Tools: []string{
					"k6/info",
					"k6/search_documentation",
					"k6/validate_script",
					"k6/run_script",
				},
				McpServers: []string{agents.K6McpServer},
			},
			TemplateName: agents.TemplateK6PlaywrightTestConverter,
			OutputPath:   paths.AgentFile("k6-playwright-test-converter.agent.md"),
		},
	}
}

// validateExistingFolders checks for existing folders and removes them if Force is true.
func (v *VSCode) validateExistingFolders(
	paths *vscodePaths,
	opts agents.InitializerOptions,
	result *agents.InitializeResult,
) error {
	if _, err := v.fileSystem.Stat(paths.AgentsDir()); err == nil {
		if !opts.Force {
			return fmt.Errorf(
				".github/agents folder already exists at %s, use --force to overwrite",
				paths.AgentsDir(),
			)
		}
		result.Warnings = append(
			result.Warnings,
			fmt.Sprintf("Removing existing .github/agents folder at %s", paths.AgentsDir()),
		)
		if err := v.fileSystem.RemoveAll(paths.AgentsDir()); err != nil {
			return fmt.Errorf("failed to remove existing .github/agents folder: %w", err)
		}
	}

	if _, err := v.fileSystem.Stat(paths.VSCodeDir()); err == nil {
		if !opts.Force {
			return fmt.Errorf(".vscode folder already exists at %s, use --force to overwrite", paths.VSCodeDir())
		}
		result.Warnings = append(result.Warnings, fmt.Sprintf("Removing existing .vscode folder at %s", paths.VSCodeDir()))
		if err := v.fileSystem.RemoveAll(paths.VSCodeDir()); err != nil {
			return fmt.Errorf("failed to remove existing .vscode folder: %w", err)
		}
	}

	return nil
}

// createGitHubStructure creates the .github folder structure.
func (v *VSCode) createGitHubStructure(paths *vscodePaths, result *agents.InitializeResult) error {
	// Create .github folder (use MkdirAll to avoid conflicts with existing .github)
	if err := v.fileSystem.MkdirAll(paths.GitHubDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create .github folder: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.GitHubDir())

	// Create agents subfolder
	if err := v.fileSystem.MkdirAll(paths.AgentsDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create .github/agents folder: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.AgentsDir())

	return nil
}

// createVSCodeStructure creates the .vscode folder structure.
func (v *VSCode) createVSCodeStructure(paths *vscodePaths, result *agents.InitializeResult) error {
	// Create .vscode folder
	if err := v.fileSystem.MkdirAll(paths.VSCodeDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create .vscode folder: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, paths.VSCodeDir())

	return nil
}

// createMcpConfig creates the MCP configuration file.
func (v *VSCode) createMcpConfig(paths *vscodePaths, result *agents.InitializeResult) error {
	// Define MCP configuration structure
	type mcpServerDef struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}

	type mcpConfig struct {
		Servers map[string]mcpServerDef `json:"servers"`
		Inputs  []any                   `json:"inputs"`
	}

	config := mcpConfig{
		Servers: map[string]mcpServerDef{
			agents.K6McpServer: {
				Type:    "stdio",
				Command: "mcp-k6",
				Args:    []string{},
			},
		},
		Inputs: []any{},
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP configuration: %w", err)
	}

	mcpPath := paths.McpConfigFile()
	if err := v.fileSystem.WriteFile(mcpPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write MCP configuration file at %s: %w", mcpPath, err)
	}

	result.FilesCreated = append(result.FilesCreated, mcpPath)
	return nil
}
