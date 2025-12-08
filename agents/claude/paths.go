package claude

import "path/filepath"

// claudePaths encapsulates all Claude-specific path logic.
type claudePaths struct {
	baseDir string
}

// newClaudePaths creates a new claudePaths instance.
func newClaudePaths(baseDir string) *claudePaths {
	return &claudePaths{baseDir: baseDir}
}

// ClaudeDir returns the path to the .claude directory.
func (p *claudePaths) ClaudeDir() string {
	return filepath.Join(p.baseDir, claudeFolderName)
}

// AgentsDir returns the path to the agents directory.
func (p *claudePaths) AgentsDir() string {
	return filepath.Join(p.ClaudeDir(), claudeAgentsFolderName)
}

// SettingsFile returns the path to the settings file.
func (p *claudePaths) SettingsFile() string {
	return filepath.Join(p.ClaudeDir(), claudeProjectSettingsFile)
}

// AgentFile returns the path to an agent file.
func (p *claudePaths) AgentFile(filename string) string {
	return filepath.Join(p.AgentsDir(), filename)
}
