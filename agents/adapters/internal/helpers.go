// Package internal provides shared helpers for adapter implementations.
package internal

import (
	"path"

	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
)

const ownerMarker = "xk6-subcommand-agent:v1"

// PlanSkillFolder creates PlannedFile entries for copying a skill folder
// into baseDir/<skill.Name>/, including SKILL.md, sibling files, and the
// .xk6-subcommand-agent-managed marker file.
func PlanSkillFolder(baseDir string, s core.Skill) ([]adapters.PlannedFile, error) {
	// Render the SKILL.md from the parsed skill.
	skillContent, err := core.RenderSkillMD(s)
	if err != nil {
		return nil, err
	}

	skillDir := path.Join(baseDir, s.Name)

	files := make([]adapters.PlannedFile, 0, 1+len(s.Files))
	files = append(files, adapters.PlannedFile{
		Path:        path.Join(skillDir, "SKILL.md"),
		Content:     skillContent,
		Mode:        adapters.CreateOnly,
		OwnerMarker: ownerMarker,
	})

	// Copy sibling files.
	for _, f := range s.Files {
		files = append(files, adapters.PlannedFile{
			Path:        path.Join(skillDir, f.RelPath),
			Content:     f.Content,
			Mode:        adapters.CreateOnly,
			OwnerMarker: ownerMarker,
		})
	}

	return files, nil
}
