// Package agents — see doc.go for the overview.
//
//nolint:forbidigo // os.FileMode is a type, not a forbidden call
package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Platform describes a single target tool (Claude Code, VS Code, ...) as
// data. Platforms are declared in [Platforms] and passed to [Install] —
// they contain no behavior beyond building the list of files to write.
type Platform struct {
	// ID is the short lowercase identifier used on the CLI
	// (`k6 x agent init <ID>`).
	ID string

	// DisplayName is the human-readable name shown in CLI output.
	DisplayName string

	// Roots are directories (relative to the project root) that this
	// platform "owns" — they are removed when --force is passed and are
	// used by [StatusPaths] to decide whether the platform is installed.
	Roots []string

	// StatusPaths lists extra paths (relative to the project root)
	// beyond Roots that signal an installed platform. Used by the
	// status command.
	StatusPaths []StatusPath

	// Files builds the concrete list of files to write for a given set
	// of shared agents. Called once per install.
	Files func(shared []AgentConfig) ([]FileSpec, error)

	// PostInstall is an optional multi-line message printed after a
	// successful install (e.g. manual GitHub Copilot setup steps).
	PostInstall string
}

// FileSpec is a single file to be written, with its path relative to the
// project root.
type FileSpec struct {
	Path    string
	Mode    os.FileMode
	Content []byte
}

// StatusPath is a project-relative path inspected by the status command.
type StatusPath struct {
	// Path is relative to the project root.
	Path string
	// Kind is either "file", "dir", or "non-empty-dir".
	Kind string
	// Description is what the status command prints when the path is
	// present (e.g. "mcp config").
	Description string
}

// Install applies the given [Platform] to the workspace described by opts.
// It enforces the --force/--dry-run contract and returns a summary of files
// it created.
func Install(ctx context.Context, fs FileSystem, p Platform, opts InitializerOptions) (*InitializeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("initialization cancelled: %w", err)
	}

	result := &InitializeResult{}

	workingDir, err := resolveWorkingDir(fs, opts.WorkingDir)
	if err != nil {
		return nil, err
	}

	files, err := p.Files(SharedAgents())
	if err != nil {
		return nil, fmt.Errorf("%s: build file list: %w", p.ID, err)
	}

	// Enforce --force on any existing root.
	for _, root := range p.Roots {
		abs := filepath.Join(workingDir, root)
		if _, err := fs.Stat(abs); err == nil {
			if !opts.Force {
				return nil, fmt.Errorf(
					"%s already exists at %s, use --force to overwrite",
					root, abs,
				)
			}
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("removing existing %s", root),
			)
		}
	}

	if opts.DryRun {
		result.Warnings = append(result.Warnings, "dry-run: no files were written")
		return result, nil
	}

	// Remove roots under --force. Do this before mkdirs so stale contents
	// don't leak into the new install.
	for _, root := range p.Roots {
		abs := filepath.Join(workingDir, root)
		if err := fs.RemoveAll(abs); err != nil {
			return nil, fmt.Errorf("remove %s: %w", abs, err)
		}
	}

	// Write files (creating parent directories as needed).
	seen := make(map[string]bool)
	for _, f := range files {
		abs := filepath.Join(workingDir, f.Path)
		dir := filepath.Dir(abs)
		if !seen[dir] {
			if err := fs.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create directory %s: %w", dir, err)
			}
			seen[dir] = true
		}
		mode := f.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := fs.WriteFile(abs, f.Content, mode); err != nil {
			return nil, fmt.Errorf("write %s: %w", abs, err)
		}
		result.FilesCreated = append(result.FilesCreated, f.Path)
	}

	return result, nil
}

func resolveWorkingDir(fs FileSystem, dir string) (string, error) {
	if strings.TrimSpace(dir) != "" {
		return dir, nil
	}
	wd, err := fs.WorkingDir()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return wd, nil
}
