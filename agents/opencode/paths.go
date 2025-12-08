package opencode

import "path/filepath"

// openCodePaths encapsulates all OpenCode-specific path logic.
type openCodePaths struct {
	baseDir string
}

// newOpenCodePaths creates a new openCodePaths instance.
func newOpenCodePaths(baseDir string) *openCodePaths {
	return &openCodePaths{baseDir: baseDir}
}

// OpenCodeDir returns the path to the .opencode directory.
func (p *openCodePaths) OpenCodeDir() string {
	return filepath.Join(p.baseDir, openCodeFolderName)
}

// PromptsDir returns the path to the prompts directory.
func (p *openCodePaths) PromptsDir() string {
	return filepath.Join(p.OpenCodeDir(), openCodePromptsFolderName)
}

// PromptFile returns the path to a prompt file.
func (p *openCodePaths) PromptFile(filename string) string {
	return filepath.Join(p.PromptsDir(), filename)
}

// ConfigFile returns the path to the opencode.json configuration file.
// The config file is placed at the project root, not inside .opencode/.
func (p *openCodePaths) ConfigFile() string {
	return filepath.Join(p.baseDir, openCodeConfigFileName)
}
