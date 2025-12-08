package initagents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const mcpBinaryName = "mcp-k6"

type StatusReport struct {
	Platforms []PlatformStatus
	MCP       MCPStatus
}

type PlatformStatus struct {
	Name      string
	Installed bool
	Details   []string
	Missing   []string
}

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
	platforms := make([]PlatformStatus, 0, len(supportedPlatforms))

	claude, err := r.claudeStatus(root)
	if err != nil {
		return nil, err
	}
	platforms = append(platforms, claude)

	vscode, err := r.vscodeStatus(root)
	if err != nil {
		return nil, err
	}
	platforms = append(platforms, vscode)

	opencode, err := r.openCodeStatus(root)
	if err != nil {
		return nil, err
	}
	platforms = append(platforms, opencode)

	mcp, err := r.mcpStatus()
	if err != nil {
		return nil, err
	}

	return &StatusReport{
		Platforms: platforms,
		MCP:       mcp,
	}, nil
}

func (r *statusReporter) claudeStatus(root string) (PlatformStatus, error) {
	status := PlatformStatus{Name: "claude"}

	claudeDir := filepath.Join(root, ".claude")
	exists, err := r.dirExists(claudeDir)
	if err != nil {
		return status, fmt.Errorf("failed to inspect %s: %w", claudeDir, err)
	}
	if !exists {
		status.Missing = append(status.Missing, ".claude directory")
		return status, nil
	}

	agentsDir := filepath.Join(claudeDir, "agents")
	count, err := r.countEntries(agentsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			status.Missing = append(status.Missing, ".claude/agents directory")
			return status, nil
		}
		return status, fmt.Errorf("failed to inspect %s: %w", agentsDir, err)
	}

	if count == 0 {
		status.Missing = append(status.Missing, ".claude/agents is empty")
		return status, nil
	}

	status.Installed = true
	status.Details = append(status.Details, fmt.Sprintf("%d agent file(s) in .claude/agents", count))

	return status, nil
}

func (r *statusReporter) vscodeStatus(root string) (PlatformStatus, error) {
	status := PlatformStatus{Name: "vscode"}

	agentsDir := filepath.Join(root, ".github", "agents")
	count, err := r.countEntries(agentsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			status.Missing = append(status.Missing, ".github/agents directory")
		} else {
			return status, fmt.Errorf("failed to inspect %s: %w", agentsDir, err)
		}
	} else if count == 0 {
		status.Missing = append(status.Missing, ".github/agents is empty")
	}

	mcpConfigPath := filepath.Join(root, ".vscode", "mcp.json")
	mcpConfigExists, err := r.fileExists(mcpConfigPath)
	if err != nil {
		return status, fmt.Errorf("failed to inspect %s: %w", mcpConfigPath, err)
	}
	if !mcpConfigExists {
		status.Missing = append(status.Missing, ".vscode/mcp.json")
	}

	if len(status.Missing) == 0 {
		status.Installed = true
		status.Details = append(status.Details, fmt.Sprintf("%d agent file(s) in .github/agents", count))
		status.Details = append(status.Details, ".vscode/mcp.json detected")
	}

	return status, nil
}

func (r *statusReporter) openCodeStatus(root string) (PlatformStatus, error) {
	status := PlatformStatus{Name: "opencode"}

	promptsDir := filepath.Join(root, ".opencode", "prompts")
	count, err := r.countEntries(promptsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			status.Missing = append(status.Missing, ".opencode/prompts directory")
		} else {
			return status, fmt.Errorf("failed to inspect %s: %w", promptsDir, err)
		}
	} else if count == 0 {
		status.Missing = append(status.Missing, ".opencode/prompts is empty")
	}

	configPath := filepath.Join(root, "opencode.json")
	configExists, err := r.fileExists(configPath)
	if err != nil {
		return status, fmt.Errorf("failed to inspect %s: %w", configPath, err)
	}
	if !configExists {
		status.Missing = append(status.Missing, "opencode.json")
	}

	if len(status.Missing) == 0 {
		status.Installed = true
		status.Details = append(status.Details, fmt.Sprintf("%d prompt file(s) in .opencode/prompts", count))
		status.Details = append(status.Details, "opencode.json detected")
	}

	return status, nil
}

func (r *statusReporter) mcpStatus() (MCPStatus, error) {
	if r.lookPath == nil {
		r.lookPath = exec.LookPath
	}

	path, err := r.lookPath(mcpBinaryName)
	if err != nil {
		var execErr *exec.Error
		if errors.Is(err, exec.ErrNotFound) || (errors.As(err, &execErr) && execErr.Err == exec.ErrNotFound) {
			return MCPStatus{Installed: false}, nil
		}
		return MCPStatus{}, fmt.Errorf("failed to inspect %s: %w", mcpBinaryName, err)
	}

	return MCPStatus{
		Installed: true,
		Path:      path,
	}, nil
}

func (r *statusReporter) dirExists(path string) (bool, error) {
	info, err := r.fs.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
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

func (r *statusReporter) countEntries(path string) (int, error) {
	entries, err := r.fs.ReadDir(path)
	if err != nil {
		return 0, err
	}

	count := 0
	for range entries {
		count++
	}
	return count, nil
}
