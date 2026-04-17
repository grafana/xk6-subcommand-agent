// Package agents provides abstractions for initializing AI coding agent
// configurations across different platforms (Claude, Copilot, Cursor, etc.).
package agents

import "embed"

// SkillsFS contains the embedded SKILL.md folders and their sibling files.
//
//go:embed skills/**
var SkillsFS embed.FS
