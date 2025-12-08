// Package agents provides abstractions for initializing AI coding agent
// configurations across different platforms (Claude, Copilot, Cursor, etc.).
//
// The package defines core interfaces and types that enable platform-agnostic
// agent initialization, while allowing platform-specific implementations to
// handle their unique requirements.
//
// Example usage:
//
//	// Initialize Claude agents
//	claudePlatform := claude.NewClaudeCode()
//	if err := agents.InitializeAgents(claudePlatform); err != nil {
//	    log.Fatal(err)
//	}
package agents

import (
	"context"
	"errors"
	"fmt"
)

// Initializer is the interface that agent platforms must implement.
type Initializer interface {
	// Initialize initializes the agent platform.
	Initialize(ctx context.Context, opts InitializerOptions) (*InitializeResult, error)

	// Name returns the platform name.
	Name() string

	// Validate checks if initialization can proceed.
	Validate(ctx context.Context, opts InitializerOptions) error
}

// InitializerOptions provides configuration for initialization.
type InitializerOptions struct {
	// WorkingDir is the base directory for initialization.
	// If empty, the current working directory will be used.
	WorkingDir string

	// Force overwrites existing files.
	Force bool

	// DryRun validates without writing files.
	DryRun bool
}

// InitializeResult provides detailed initialization results.
type InitializeResult struct {
	// FilesCreated lists all files created.
	FilesCreated []string

	// FilesUpdated lists all files updated.
	FilesUpdated []string

	// Warnings contains non-fatal issues.
	Warnings []string
}

// InitializationError wraps initialization errors with context.
type InitializationError struct {
	AgentName string
	Err       error
}

func (e *InitializationError) Error() string {
	return fmt.Sprintf("agent %q initialization failed: %v", e.AgentName, e.Err)
}

func (e *InitializationError) Unwrap() error {
	return e.Err
}

// InitializeStrategy controls error handling behavior.
type InitializeStrategy int

const (
	// StrategyFailFast stops on first error.
	StrategyFailFast InitializeStrategy = iota

	// StrategyAccumulate continues and collects all errors.
	StrategyAccumulate
)

// InitializeAgents initializes all agents with accumulate strategy.
// This is the backward-compatible function that uses context.Background().
func InitializeAgents(initializers ...Initializer) error {
	return InitializeAgentsWithContext(context.Background(), initializers...)
}

// InitializeAgentsWithContext initializes agents with context support.
func InitializeAgentsWithContext(ctx context.Context, initializers ...Initializer) error {
	opts := InitializerOptions{}
	return InitializeAgentsWithStrategy(ctx, StrategyAccumulate, opts, initializers...)
}

// InitializeAgentsWithStrategy initializes agents with configurable error handling.
func InitializeAgentsWithStrategy(
	ctx context.Context,
	strategy InitializeStrategy,
	opts InitializerOptions,
	initializers ...Initializer,
) error {
	var errs []error

	for _, initializer := range initializers {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("initialization cancelled: %w", err)
		}

		// Validate before initializing
		if err := initializer.Validate(ctx, opts); err != nil {
			wrappedErr := &InitializationError{
				AgentName: initializer.Name(),
				Err:       fmt.Errorf("validation failed: %w", err),
			}

			if strategy == StrategyFailFast {
				return wrappedErr
			}
			errs = append(errs, wrappedErr)
			continue
		}

		// Initialize
		_, err := initializer.Initialize(ctx, opts)
		if err != nil {
			wrappedErr := &InitializationError{
				AgentName: initializer.Name(),
				Err:       err,
			}

			if strategy == StrategyFailFast {
				return wrappedErr
			}
			errs = append(errs, wrappedErr)
		}
	}

	return errors.Join(errs...)
}

// InitializeAgentSpecs processes multiple agent specifications and writes agent files.
// This is a shared helper that both Claude and VSCode can use.
func InitializeAgentSpecs(
	ctx context.Context,
	formatter ConfigurationFormatter,
	renderer *TemplateRenderer,
	fs FileSystem,
	specs []AgentSpec,
	result *InitializeResult,
) error {
	for _, spec := range specs {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("initialization cancelled while creating agent %q: %w", spec.Config.Name, err)
		}

		// Validate the configuration
		if err := spec.Config.Validate(); err != nil {
			return fmt.Errorf("invalid configuration for agent %q: %w", spec.Config.Name, err)
		}

		// Format the configuration header
		header, err := formatter.FormatAgentConfig(spec.Config)
		if err != nil {
			return fmt.Errorf("failed to format configuration for agent %q: %w", spec.Config.Name, err)
		}

		// Render the template
		data := agentTemplateData{ConfigHeader: string(header)}
		content, err := renderer.Render(spec.TemplateName, data)
		if err != nil {
			return fmt.Errorf("failed to render template %q for agent %q: %w", spec.TemplateName, spec.Config.Name, err)
		}

		// Write the agent file
		if err := fs.WriteFile(spec.OutputPath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write agent file %q: %w", spec.OutputPath, err)
		}

		result.FilesCreated = append(result.FilesCreated, spec.OutputPath)
	}

	return nil
}
