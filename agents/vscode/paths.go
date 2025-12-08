package vscode

import "path/filepath"

// vscodePaths encapsulates all VSCode-specific path logic.
type vscodePaths struct {
	baseDir string
}

// newVSCodePaths creates a new vscodePaths instance.
func newVSCodePaths(baseDir string) *vscodePaths {
	return &vscodePaths{baseDir: baseDir}
}

// GitHubDir returns the path to the .github directory.
func (p *vscodePaths) GitHubDir() string {
	return filepath.Join(p.baseDir, githubFolderName)
}

// AgentsDir returns the path to the .github/agents directory.
func (p *vscodePaths) AgentsDir() string {
	return filepath.Join(p.GitHubDir(), agentsFolderName)
}

// VSCodeDir returns the path to the .vscode directory.
func (p *vscodePaths) VSCodeDir() string {
	return filepath.Join(p.baseDir, vscodeFolderName)
}

// McpConfigFile returns the path to the mcp.json file.
func (p *vscodePaths) McpConfigFile() string {
	return filepath.Join(p.VSCodeDir(), mcpConfigFileName)
}

// AgentFile returns the path to an agent file.
func (p *vscodePaths) AgentFile(filename string) string {
	return filepath.Join(p.AgentsDir(), filename)
}
