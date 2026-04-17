// Package agents — see doc.go for the overview.
package agents

// InitializerOptions configures a single platform install.
type InitializerOptions struct {
	// WorkingDir is the project root to scaffold into. If empty, the
	// current working directory (resolved via [FileSystem.WorkingDir]) is
	// used.
	WorkingDir string

	// Force removes the platform's owned roots before re-creating them.
	// Without Force, installing over an existing setup returns an error.
	Force bool

	// DryRun validates that an install could proceed but writes no files.
	DryRun bool
}

// InitializeResult reports what an install did.
type InitializeResult struct {
	// FilesCreated lists the paths (files and directories) written by the
	// install, in creation order.
	FilesCreated []string

	// Warnings contains non-fatal messages surfaced to the user.
	Warnings []string
}
