package core_test

import (
	"testing"
	"testing/fstest"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
)

func TestParseSkillMD_Valid(t *testing.T) {
	t.Parallel()

	input := []byte(`---
name: k6-load-test
description: Generate k6 load tests for APIs and services.
---

You are a senior k6 performance engineer.
`)

	skill, err := core.ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill.Name != "k6-load-test" {
		t.Errorf("expected name %q, got %q", "k6-load-test", skill.Name)
	}

	if skill.Description != "Generate k6 load tests for APIs and services." {
		t.Errorf("unexpected description: %q", skill.Description)
	}

	if skill.Body != "You are a senior k6 performance engineer.\n" {
		t.Errorf("unexpected body: %q", skill.Body)
	}

	if skill.Frontmatter["name"] != "k6-load-test" {
		t.Errorf("frontmatter name not preserved")
	}
}

func TestParseSkillMD_MissingName(t *testing.T) {
	t.Parallel()

	input := []byte(`---
description: Some description here.
---

Body text.
`)

	_, err := core.ParseSkillMD(input)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseSkillMD_MissingDescription(t *testing.T) {
	t.Parallel()

	input := []byte(`---
name: my-skill
---

Body text.
`)

	_, err := core.ParseSkillMD(input)
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestParseSkillMD_NoFrontmatter(t *testing.T) {
	t.Parallel()

	input := []byte(`Just some markdown without frontmatter.`)

	_, err := core.ParseSkillMD(input)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseSkillMD_UnclosedFrontmatter(t *testing.T) {
	t.Parallel()

	input := []byte(`---
name: test
description: test description
`)

	_, err := core.ParseSkillMD(input)
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter")
	}
}

func TestParseSkillMD_InvalidKebabCase(t *testing.T) {
	t.Parallel()

	input := []byte(`---
name: My Skill
description: Some description for the skill.
---

Body.
`)

	_, err := core.ParseSkillMD(input)
	if err == nil {
		t.Fatal("expected error for non-kebab-case name")
	}
}

func TestParseSkillMD_MultilineDescription(t *testing.T) {
	t.Parallel()

	input := []byte(`---
name: k6-test
description: >-
  A multi-line description that spans
  multiple lines in YAML.
---

Body.
`)

	skill, err := core.ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill.Description != "A multi-line description that spans multiple lines in YAML." {
		t.Errorf("unexpected description: %q", skill.Description)
	}
}

func TestLoadSkills_FromEmbedded(t *testing.T) {
	t.Parallel()

	skills, err := core.LoadSkills(agents.SkillsFS)
	if err != nil {
		t.Fatalf("failed to load embedded skills: %v", err)
	}

	if len(skills) != 11 {
		t.Fatalf("expected 11 skills, got %d", len(skills))
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true

		if s.Source == "" {
			t.Errorf("skill %q has empty Source", s.Name)
		}

		if s.Body == "" {
			t.Errorf("skill %q has empty Body", s.Name)
		}
	}

	// Verify the planner skill has its references/ sibling file.
	planner := findSkill(skills, "k6-test-planner")
	if planner == nil {
		t.Fatal("k6-test-planner skill not found")
	}

	if len(planner.Files) == 0 {
		t.Error("k6-test-planner should have sibling files (references/)")
	}

	if ref := findSkillFile(planner.Files, "references/test-types.md"); ref == nil {
		t.Error("k6-test-planner missing references/test-types.md")
	} else if len(ref.Content) == 0 {
		t.Error("test-types.md should not be empty")
	}

	expected := []string{
		"k6-test-planner",
		"k6-load-test",
		"k6-browser-test",
		"k6-smoke-test",
		"k6-playwright-converter",
		"k6-perf-test-website",
		"k6-manage",
		"k6-docs",
		"k6-cloud-investigate-test",
		"k6-trend-analysis",
		"k6-test-maintenance",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected skill %q not found", name)
		}
	}
}

func TestLoadSkills_WithSiblingFiles(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: my-skill
description: A test skill with sibling files.
---

Body.
`),
		},
		"skills/my-skill/scripts/helper.js": &fstest.MapFile{
			Data: []byte(`export function helper() {}`),
		},
	}

	skills, err := core.LoadSkills(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	if len(skills[0].Files) != 1 {
		t.Fatalf("expected 1 sibling file, got %d", len(skills[0].Files))
	}

	if skills[0].Files[0].RelPath != "scripts/helper.js" {
		t.Errorf("unexpected sibling file path: %q", skills[0].Files[0].RelPath)
	}
}

func TestLoadSkills_WithOverrides(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: my-skill
description: A test skill with overrides.
---

Body.
`),
		},
		"skills/my-skill/overrides.yaml": &fstest.MapFile{
			Data: []byte(`cursor:
  globs:
    - "**/*.js"
  alwaysApply: true
`),
		},
	}

	skills, err := core.LoadSkills(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	if skills[0].Overrides == nil {
		t.Fatal("expected overrides to be parsed")
	}

	if skills[0].Overrides.Cursor == nil {
		t.Fatal("expected cursor overrides")
	}

	if len(skills[0].Overrides.Cursor.Globs) != 1 || skills[0].Overrides.Cursor.Globs[0] != "**/*.js" {
		t.Errorf("unexpected cursor globs: %v", skills[0].Overrides.Cursor.Globs)
	}

	if !skills[0].Overrides.Cursor.AlwaysApply {
		t.Error("expected alwaysApply to be true")
	}
}

func TestLoadSkills_SkipsNonSkillDirs(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"skills/valid-skill/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: valid-skill
description: A valid skill for testing purposes.
---

Body.
`),
		},
		"skills/not-a-skill/readme.txt": &fstest.MapFile{
			Data: []byte(`This directory has no SKILL.md`),
		},
	}

	skills, err := core.LoadSkills(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	if skills[0].Name != "valid-skill" {
		t.Errorf("expected skill name %q, got %q", "valid-skill", skills[0].Name)
	}
}

func TestRenderSkillMD(t *testing.T) {
	t.Parallel()

	skill := core.Skill{
		Name:        "test-skill",
		Description: "A test skill for rendering.",
		Body:        "Some body content.\n",
		Frontmatter: map[string]any{
			"name":        "test-skill",
			"description": "A test skill for rendering.",
		},
	}

	data, err := core.RenderSkillMD(skill)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Re-parse and verify roundtrip.
	parsed, err := core.ParseSkillMD(data)
	if err != nil {
		t.Fatalf("failed to re-parse rendered SKILL.md: %v", err)
	}

	if parsed.Name != skill.Name {
		t.Errorf("name mismatch: %q vs %q", parsed.Name, skill.Name)
	}

	if parsed.Description != skill.Description {
		t.Errorf("description mismatch: %q vs %q", parsed.Description, skill.Description)
	}
}

func findSkill(skills []core.Skill, name string) *core.Skill {
	for i := range skills {
		if skills[i].Name == name {
			return &skills[i]
		}
	}

	return nil
}

func findSkillFile(files []core.SkillFile, relPath string) *core.SkillFile {
	for i := range files {
		if files[i].RelPath == relPath {
			return &files[i]
		}
	}

	return nil
}
