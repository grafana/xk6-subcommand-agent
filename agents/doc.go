// Package agents provides a data-driven way to scaffold AI agent
// configurations for multiple coding assistants (Claude Code, GitHub
// Copilot / VS Code, OpenCode, ...).
//
// The package exposes:
//
//   - A slice of [Platform] descriptors — one per target tool — built from
//     data rather than per-platform Go packages.
//   - A single [Install] function that applies a [Platform] to a workspace.
//   - Shared constants (the MCP server name, agent descriptions) and a
//     [SharedAgents] list that every platform reuses.
//
// Adding a new target (Cursor, Zed, Windsurf, ...) is a matter of appending
// another [Platform] literal to [Platforms] — no new package required.
package agents
