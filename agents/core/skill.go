// Package core provides shared types and helpers for the xk6-agent skill
// and adapter system.
package core

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a portable SKILL.md folder: frontmatter metadata,
// a markdown body, and optional sibling files.
type Skill struct {
	// Name is the kebab-case identifier from frontmatter.
	Name string
	// Description drives host-agent activation; from frontmatter.
	Description string
	// Body is the markdown content after the frontmatter.
	Body string
	// Frontmatter is the raw parsed YAML frontmatter, preserved for
	// adapters that need to pass it through.
	Frontmatter map[string]any
	// Files holds optional sibling files (scripts/, references/, etc.)
	// that are copied alongside the SKILL.md.
	Files []SkillFile
	// Source is the embedded path for diagnostics.
	Source string
	// Overrides contains optional per-target hints from overrides.yaml.
	Overrides *SkillOverrides
}

// SkillFile represents a file that lives alongside SKILL.md in the
// skill folder.
type SkillFile struct {
	// RelPath is relative to the skill folder root.
	RelPath string
	// Content is the raw file bytes.
	Content []byte
}

// SkillOverrides holds per-target configuration hints that are kept OUT
// of SKILL.md frontmatter to preserve portability. Adapters consult
// these only if relevant; absent ones are ignored.
type SkillOverrides struct {
	Cursor  *CursorOverride  `yaml:"cursor,omitempty"`
	Copilot *CopilotOverride `yaml:"copilot,omitempty"`
	Cline   *ClineOverride   `yaml:"cline,omitempty"`
}

// CursorOverride holds Cursor-specific hints.
type CursorOverride struct {
	Globs       []string `yaml:"globs,omitempty"`
	AlwaysApply bool     `yaml:"alwaysApply,omitempty"`
}

// CopilotOverride holds GitHub Copilot-specific hints.
type CopilotOverride struct {
	ApplyTo []string `yaml:"applyTo,omitempty"`
}

// ClineOverride holds Cline-specific hints.
type ClineOverride struct{}

var kebabCaseRe = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`)

// ParseSkillMD parses a SKILL.md file into a Skill. The input must contain
// YAML frontmatter delimited by --- lines, followed by the markdown body.
func ParseSkillMD(data []byte) (Skill, error) {
	content := string(data)

	// Trim leading whitespace / BOM.
	content = strings.TrimLeft(content, "\xef\xbb\xbf \t\n\r")

	if !strings.HasPrefix(content, "---") {
		return Skill{}, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Find the closing ---.
	rest := content[3:]
	fmRaw, body, found := strings.Cut(rest, "\n---")
	if !found {
		return Skill{}, fmt.Errorf("SKILL.md frontmatter is not closed (missing closing ---)")
	}

	body = strings.TrimLeft(body, "\r\n")

	// Parse YAML frontmatter.
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return Skill{}, fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}

	name, _ := fm["name"].(string)
	if name == "" {
		return Skill{}, fmt.Errorf("SKILL.md frontmatter must contain a non-empty 'name' field")
	}

	if !kebabCaseRe.MatchString(name) {
		return Skill{}, fmt.Errorf("skill name must be kebab-case: got %q", name)
	}

	description, _ := fm["description"].(string)
	if description == "" {
		return Skill{}, fmt.Errorf("SKILL.md frontmatter must contain a non-empty 'description' field")
	}

	return Skill{
		Name:        name,
		Description: description,
		Body:        body,
		Frontmatter: fm,
	}, nil
}

// LoadSkills walks an fs.FS rooted at "skills/" and returns all parsed
// skills. Each subdirectory containing a SKILL.md is treated as one skill.
func LoadSkills(fsys fs.FS) ([]Skill, error) {
	entries, err := fs.ReadDir(fsys, "skills")
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []Skill

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := path.Join("skills", entry.Name())
		skillFile := path.Join(skillDir, "SKILL.md")

		data, err := fs.ReadFile(fsys, skillFile)
		if err != nil {
			continue // directory without SKILL.md — skip
		}

		skill, err := ParseSkillMD(data)
		if err != nil {
			return nil, fmt.Errorf("skill %q: %w", entry.Name(), err)
		}

		skill.Source = skillFile

		// Collect sibling files.
		skill.Files, err = collectSiblingFiles(fsys, skillDir)
		if err != nil {
			return nil, fmt.Errorf("skill %q: failed to collect sibling files: %w", entry.Name(), err)
		}

		// Parse overrides.yaml if present.
		overridesPath := path.Join(skillDir, "overrides.yaml")
		if overridesData, readErr := fs.ReadFile(fsys, overridesPath); readErr == nil {
			var overrides SkillOverrides
			if yamlErr := yaml.Unmarshal(overridesData, &overrides); yamlErr != nil {
				return nil, fmt.Errorf("skill %q: failed to parse overrides.yaml: %w", entry.Name(), yamlErr)
			}

			skill.Overrides = &overrides
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// collectSiblingFiles walks a skill directory and returns all files except
// SKILL.md and overrides.yaml, with paths relative to the skill dir.
func collectSiblingFiles(fsys fs.FS, skillDir string) ([]SkillFile, error) {
	var files []SkillFile

	err := fs.WalkDir(fsys, skillDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if name == "SKILL.md" || name == "overrides.yaml" {
			return nil
		}

		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", p, readErr)
		}

		relPath, relErr := relativePath(skillDir, p)
		if relErr != nil {
			return relErr
		}

		files = append(files, SkillFile{
			RelPath: relPath,
			Content: data,
		})

		return nil
	})

	return files, err
}

// relativePath returns p relative to base, using forward slashes.
func relativePath(base, p string) (string, error) {
	if !strings.HasPrefix(p, base+"/") {
		return "", fmt.Errorf("path %q is not under base %q", p, base)
	}

	return p[len(base)+1:], nil
}

// RenderSkillMD reconstructs a complete SKILL.md file from a Skill's
// frontmatter and body. This is used by adapters that copy skills verbatim.
func RenderSkillMD(s Skill) ([]byte, error) {
	fmBytes, err := yaml.Marshal(s.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skill frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(s.Body)

	return buf.Bytes(), nil
}
