package agents_test

import (
	"io/fs"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents"
)

func TestSkillsFS_ContainsExpectedSkills(t *testing.T) {
	t.Parallel()

	expectedSkills := []string{
		"skills/k6-test-planner/SKILL.md",
		"skills/k6-load-test/SKILL.md",
		"skills/k6-browser-test/SKILL.md",
		"skills/k6-smoke-test/SKILL.md",
		"skills/k6-playwright-converter/SKILL.md",
		"skills/k6-perf-test-website/SKILL.md",
		"skills/k6-test-maintenance/SKILL.md",
	}

	for _, path := range expectedSkills {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			data, err := fs.ReadFile(agents.SkillsFS, path)
			if err != nil {
				t.Fatalf("expected embedded file %q to exist: %v", path, err)
			}
			if len(data) == 0 {
				t.Fatalf("expected embedded file %q to be non-empty", path)
			}
		})
	}
}

func TestSkillsFS_SkillCount(t *testing.T) {
	t.Parallel()

	var count int
	err := fs.WalkDir(agents.SkillsFS, "skills", func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "SKILL.md" {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk embedded skills FS: %v", err)
	}

	if count != 7 {
		t.Fatalf("expected 7 SKILL.md files, got %d", count)
	}
}
