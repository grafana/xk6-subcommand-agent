// Package initagents — see register.go for the command-wiring entry point.
package initagents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grafana/xk6-agent/agents"
)

const mcpBinaryName = "mcp-k6"

// StatusReport captures the output of `k6 x agent status`.
type StatusReport struct {
	Platforms []PlatformStatus
	MCP       MCPStatus
}

// PlatformStatus describes one platform's installation state.
type PlatformStatus struct {
	// ID is the lowercase platform identifier used by the CLI.
	ID string
	// DisplayName is the human-readable platform name.
	DisplayName string
	// Installed is true when every required path exists.
	Installed bool
	// Details are human-readable lines shown under an installed platform.
	Details []string
	// Missing are human-readable lines shown under a not-installed platform.
	Missing []string
}

// MCPStatus reports whether the mcp-k6 binary is resolvable on PATH.
type MCPStatus struct {
	Installed bool
	Path      string
}

type statusCollector interface {
	Collect(root string) (*StatusReport, error)
}

type statusReporter struct {
	fs       fileInspector
	lookPath func(string) (string, error)
}

type fileInspector interface {
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

type osFileInspector struct{}

func (osFileInspector) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path) //nolint:forbidigo // filesystem seam
}

func (osFileInspector) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path) //nolint:forbidigo // filesystem seam
}

func newStatusReporter() *statusReporter {
	return &statusReporter{
		fs:       osFileInspector{},
		lookPath: exec.LookPath,
	}
}

// Collect walks every platform descriptor and produces a StatusReport.
func (r *statusReporter) Collect(root string) (*StatusReport, error) {
	platforms := make([]PlatformStatus, 0, len(agents.Platforms))
	for _, p := range agents.Platforms {
		ps, err := r.platformStatus(root, p)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, ps)
	}
	mcp, err := r.mcpStatus()
	if err != nil {
		return nil, err
	}
	return &StatusReport{Platforms: platforms, MCP: mcp}, nil
}

// platformStatus evaluates every StatusPath a platform declares and
// summarizes the result. A platform is considered "installed" only if
// every declared path is present (and non-empty, for non-empty-dir paths).
func (r *statusReporter) platformStatus(root string, p agents.Platform) (PlatformStatus, error) {
	status := PlatformStatus{ID: p.ID, DisplayName: p.DisplayName}

	if len(p.StatusPaths) == 0 {
		// Nothing to check — treat as never installed.
		return status, nil
	}

	allPresent := true
	for _, sp := range p.StatusPaths {
		abs := filepath.Join(root, sp.Path)
		present, detail, err := r.checkStatusPath(abs, sp)
		if err != nil {
			return status, err
		}
		if !present {
			allPresent = false
			status.Missing = append(status.Missing, sp.Path)
			continue
		}
		if detail != "" {
			status.Details = append(status.Details, detail)
		}
	}

	status.Installed = allPresent
	return status, nil
}

func (r *statusReporter) checkStatusPath(abs string, sp agents.StatusPath) (bool, string, error) {
	switch sp.Kind {
	case "file":
		info, err := r.fs.Stat(abs)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, "", nil
			}
			return false, "", fmt.Errorf("inspect %s: %w", abs, err)
		}
		if info.IsDir() {
			return false, "", nil
		}
		return true, fmt.Sprintf("%s (%s)", sp.Path, sp.Description), nil

	case "dir":
		info, err := r.fs.Stat(abs)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, "", nil
			}
			return false, "", fmt.Errorf("inspect %s: %w", abs, err)
		}
		if !info.IsDir() {
			return false, "", nil
		}
		return true, fmt.Sprintf("%s (%s)", sp.Path, sp.Description), nil

	case "non-empty-dir":
		entries, err := r.fs.ReadDir(abs)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, "", nil
			}
			return false, "", fmt.Errorf("inspect %s: %w", abs, err)
		}
		if len(entries) == 0 {
			return false, "", nil
		}
		return true, fmt.Sprintf("%d %s in %s", len(entries), sp.Description, sp.Path), nil

	default:
		return false, "", fmt.Errorf("unknown status path kind %q", sp.Kind)
	}
}

func (r *statusReporter) mcpStatus() (MCPStatus, error) {
	if r.lookPath == nil {
		r.lookPath = exec.LookPath
	}
	path, err := r.lookPath(mcpBinaryName)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return MCPStatus{Installed: false}, nil
		}
		return MCPStatus{}, fmt.Errorf("inspect %s: %w", mcpBinaryName, err)
	}
	return MCPStatus{Installed: true, Path: path}, nil
}
