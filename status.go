//nolint:forbidigo
package initagents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
)

// StatusReport holds the overall status of agent installations.
type StatusReport struct {
	Platforms []PlatformStatus
	MCP       MCPStatus
}

// PlatformStatus holds the status of a single platform.
type PlatformStatus struct {
	Name      string
	Installed bool
	Details   []string
	Missing   []string
}

// MCPStatus holds the status of the MCP binary.
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
	return os.Stat(path)
}

func (osFileInspector) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func newStatusReporter() *statusReporter {
	return &statusReporter{
		fs:       osFileInspector{},
		lookPath: exec.LookPath,
	}
}

func (r *statusReporter) Collect(root string) (*StatusReport, error) {
	allTargets := adapters.All()
	platforms := make([]PlatformStatus, 0, len(allTargets))

	for _, t := range allTargets {
		status, err := r.targetStatus(root, t)
		if err != nil {
			return nil, err
		}

		platforms = append(platforms, status)
	}

	mcpStatus, err := r.mcpStatus()
	if err != nil {
		return nil, err
	}

	return &StatusReport{
		Platforms: platforms,
		MCP:       mcpStatus,
	}, nil
}

// targetStatus checks whether a target's files are installed by looking
// for its MCP config path and skills directory.
func (r *statusReporter) targetStatus(root string, t adapters.Target) (PlatformStatus, error) {
	status := PlatformStatus{Name: t.Name()}
	caps := t.Capabilities()

	// Check for the MCP config file if the target defines one.
	if caps.MCPConfigPath != "" {
		mcpPath := filepath.Join(root, caps.MCPConfigPath)
		exists, err := r.fileExists(mcpPath)
		if err != nil {
			return status, fmt.Errorf("failed to inspect %s: %w", mcpPath, err)
		}

		if exists {
			status.Details = append(status.Details, fmt.Sprintf("%s detected", caps.MCPConfigPath))
		} else {
			status.Missing = append(status.Missing, caps.MCPConfigPath)
		}
	}

	// If nothing is missing, consider it installed.
	status.Installed = len(status.Missing) == 0 && len(status.Details) > 0

	return status, nil
}

func (r *statusReporter) mcpStatus() (MCPStatus, error) {
	if r.lookPath == nil {
		r.lookPath = exec.LookPath
	}

	path, err := r.lookPath("k6")
	if err != nil {
		var execErr *exec.Error
		if errors.Is(err, exec.ErrNotFound) || (errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)) {
			return MCPStatus{Installed: false}, nil
		}

		return MCPStatus{}, fmt.Errorf("failed to inspect %s: %w", "k6", err)
	}

	return MCPStatus{
		Installed: true,
		Path:      path,
	}, nil
}

func (r *statusReporter) fileExists(path string) (bool, error) {
	info, err := r.fs.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return !info.IsDir(), nil
}
