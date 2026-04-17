// Package agents — see doc.go for the overview.
package agents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Frontmatter describes the YAML header that platforms prepend to agent
// Markdown files. Zero-value fields are omitted from the output.
type Frontmatter struct {
	Name        string
	Description string
	Model       string
	// Tools is the list of tool names granted to the agent. Emitted as a
	// YAML flow sequence when empty ("tools: []") and as a block sequence
	// otherwise.
	Tools []string
	// MCPServers, if non-empty, is emitted as a nested YAML mapping under
	// the "mcp-servers:" key. Platforms that store MCP config out-of-band
	// (e.g. Claude's settings.local.json) leave this nil.
	MCPServers map[string]MCPServerSpec
}

// MCPServerSpec is the subset of MCP server fields the frontmatter emitter
// knows how to render. Platforms that need additional fields should emit
// them through their own config file rather than via frontmatter.
type MCPServerSpec struct {
	Type    string
	Command string
	Args    []string
	Tools   []string
}

// Render produces the `---`-delimited YAML frontmatter bytes, including the
// trailing newline. The emitter is intentionally minimal: it handles the
// exact shapes this extension needs and nothing else.
func (f Frontmatter) Render() []byte {
	var b bytes.Buffer
	b.WriteString("---\n")
	if f.Name != "" {
		fmt.Fprintf(&b, "name: %s\n", yamlString(f.Name))
	}
	if f.Description != "" {
		writeYAMLDescription(&b, f.Description)
	}
	if f.Model != "" {
		fmt.Fprintf(&b, "model: %s\n", yamlString(f.Model))
	}
	writeYAMLStringList(&b, "tools", f.Tools)
	if len(f.MCPServers) > 0 {
		writeYAMLMCPServers(&b, f.MCPServers)
	}
	b.WriteString("---\n")
	return b.Bytes()
}

// writeYAMLDescription emits a single- or multi-line description. Multi-line
// strings use a literal block scalar (`|`) to preserve newlines exactly.
func writeYAMLDescription(b *bytes.Buffer, description string) {
	if !strings.ContainsRune(description, '\n') {
		fmt.Fprintf(b, "description: %s\n", yamlString(description))
		return
	}
	b.WriteString("description: |\n")
	// Normalize line endings and indent each line by two spaces.
	lines := strings.Split(strings.ReplaceAll(description, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		fmt.Fprintf(b, "  %s\n", line)
	}
}

func writeYAMLStringList(b *bytes.Buffer, key string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(b, "%s: []\n", key)
		return
	}
	fmt.Fprintf(b, "%s:\n", key)
	for _, v := range values {
		fmt.Fprintf(b, "  - %s\n", yamlString(v))
	}
}

func writeYAMLMCPServers(b *bytes.Buffer, servers map[string]MCPServerSpec) {
	// Stable output: iterate keys in sorted order.
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	b.WriteString("mcp-servers:\n")
	for _, name := range names {
		spec := servers[name]
		fmt.Fprintf(b, "  %s:\n", yamlString(name))
		if spec.Type != "" {
			fmt.Fprintf(b, "    type: %s\n", yamlString(spec.Type))
		}
		if spec.Command != "" {
			fmt.Fprintf(b, "    command: %s\n", yamlString(spec.Command))
		}
		switch {
		case spec.Args == nil:
			// Caller didn't specify args; omit the key entirely.
		case len(spec.Args) == 0:
			b.WriteString("    args: []\n")
		default:
			b.WriteString("    args:\n")
			for _, a := range spec.Args {
				fmt.Fprintf(b, "      - %s\n", yamlString(a))
			}
		}
		if spec.Tools == nil {
			continue
		}
		if len(spec.Tools) == 0 {
			b.WriteString("    tools: []\n")
			continue
		}
		b.WriteString("    tools:\n")
		for _, t := range spec.Tools {
			fmt.Fprintf(b, "      - %s\n", yamlString(t))
		}
	}
}

// yamlString returns a YAML-safe scalar representation of s. If s needs
// quoting (contains YAML-reserved characters, looks boolean-like, is empty,
// or starts/ends with whitespace) it is rendered as a double-quoted string
// with standard escapes. Otherwise it is emitted bare.
func yamlString(s string) string {
	if s == "" {
		return `""`
	}
	if needsYAMLQuoting(s) {
		return yamlDoubleQuote(s)
	}
	return s
}

func needsYAMLQuoting(s string) bool {
	// Leading/trailing whitespace is always ambiguous.
	if s != strings.TrimSpace(s) {
		return true
	}
	// Values that YAML would parse as non-strings.
	switch strings.ToLower(s) {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		return true
	}
	// Leading characters that start a YAML indicator.
	switch s[0] {
	case '&', '*', '!', '|', '>', '%', '@', '`', '"', '\'', '#', '[', ']', '{', '}', ',', '?', ':', '-':
		return true
	}
	// Any control character forces quoting.
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
		if r == '#' || r == ':' {
			// Contextually ambiguous in YAML flow/block scalars.
			return true
		}
	}
	return false
}

func yamlDoubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
