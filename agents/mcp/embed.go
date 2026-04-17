// Package mcp provides the canonical MCP server definitions embedded
// from servers.yaml.
package mcp

import _ "embed"

// ServersYAML contains the raw bytes of the canonical MCP server schema.
//
//go:embed servers.yaml
var ServersYAML []byte
