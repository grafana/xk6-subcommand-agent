// Package adapters defines the Target interface and registry for xk6-agent
// target adapters. Each supported AI coding tool (Claude Code, Cursor, etc.)
// implements Target and self-registers via init().
package adapters

import (
	"context"

	"github.com/grafana/xk6-agent/agents/core"
)

// Target is the interface that every adapter must implement.
// Plan() is pure computation — it must not touch the filesystem.
type Target interface {
	// Name is the canonical CLI name, e.g. "claude-code".
	Name() string

	// DisplayName is human-readable, e.g. "Claude Code".
	DisplayName() string

	// Capabilities reports what this target supports natively.
	Capabilities() Capabilities

	// Plan returns the set of files this target wants to write/modify
	// for the given inputs. It does NOT touch the filesystem.
	Plan(ctx context.Context, in Inputs) (Plan, error)
}

// Capabilities describes what a target supports natively.
type Capabilities struct {
	// NativeSkills is true if the target consumes SKILL.md verbatim.
	NativeSkills bool
	// MCPConfigPath is the relative path of the MCP config file
	// (e.g. ".mcp.json", ".vscode/mcp.json"). Empty if not applicable.
	MCPConfigPath string
}

// Inputs bundles everything an adapter needs to produce a Plan.
type Inputs struct {
	Skills []core.Skill
	MCP    core.MCPConfig
	Root   string // user's project root
}

// Plan is the output of Target.Plan(). It describes what files to
// create, update, or merge — without touching the filesystem.
type Plan struct {
	Files   []PlannedFile
	Notices []string // human-readable notes for the CLI
}

// PlannedFile describes a single file operation.
type PlannedFile struct {
	// Path is relative to Inputs.Root.
	Path string
	// Content is the desired file content.
	Content []byte
	// Mode controls how the file is written.
	Mode WriteMode
	// MergeKey is the dot-path for MergeJSONByKey mode
	// (e.g. "mcpServers.k6").
	MergeKey string
	// OwnerMarker identifies managed files (e.g. "xk6-agent:v1").
	OwnerMarker string
}

// WriteMode controls how a PlannedFile is applied to the filesystem.
type WriteMode int

const (
	// CreateOnly creates the file if it doesn't exist, or overwrites
	// if it is managed by us (marker present).
	CreateOnly WriteMode = iota
	// OverwriteIfManaged overwrites only if the marker says we own it.
	OverwriteIfManaged
	// MergeJSONByKey does a surgical JSON merge at the specified MergeKey.
	MergeJSONByKey
)
