//nolint:forbidigo,gosec
package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/xk6-subcommand-agent/agents/core"
)

func TestApply_CreateOnly_NewFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	files := []core.PlannedFile{
		{Path: "test/hello.txt", Content: []byte("hello"), Mode: core.CreateOnlyMode},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outcomes) != 1 || outcomes[0].Status != core.Created {
		t.Fatalf("expected Created, got %v", outcomes)
	}

	data, err := os.ReadFile(filepath.Join(root, "test/hello.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestApply_CreateOnly_ExistingSkipped(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Pre-create the file.
	dir := filepath.Join(root, "test")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	files := []core.PlannedFile{
		{Path: "test/hello.txt", Content: []byte("new"), Mode: core.CreateOnlyMode},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outcomes[0].Status != core.Skipped {
		t.Errorf("expected Skipped, got %v", outcomes[0].Status)
	}

	// Original content should be preserved.
	data, err := os.ReadFile(filepath.Join(root, "test/hello.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "original" {
		t.Errorf("expected original content, got %q", string(data))
	}
}

func TestApply_CreateOnly_ExistingForced(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	dir := filepath.Join(root, "test")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	files := []core.PlannedFile{
		{Path: "test/hello.txt", Content: []byte("new"), Mode: core.CreateOnlyMode},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{Force: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outcomes[0].Status != core.Updated {
		t.Errorf("expected Updated, got %v", outcomes[0].Status)
	}

	data, err := os.ReadFile(filepath.Join(root, "test/hello.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "new" {
		t.Errorf("expected new content, got %q", string(data))
	}
}

func TestApply_MergeJSON_NewFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := []byte(`{"mcpServers": {"k6": {"command": "k6"}}}`)

	files := []core.PlannedFile{
		{
			Path:     ".mcp.json",
			Content:  content,
			Mode:     core.MergeJSONByKeyMode,
			MergeKey: "mcpServers.k6",
		},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outcomes[0].Status != core.Created {
		t.Errorf("expected Created, got %v", outcomes[0].Status)
	}
}

func TestApply_MergeJSON_PreservesOtherKeys(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Pre-create with existing content.
	existing := []byte(`{
  "mcpServers": {
    "other": {"command": "other-cmd"}
  }
}`)

	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), existing, 0o600); err != nil {
		t.Fatal(err)
	}

	newContent := []byte(`{"mcpServers": {"k6": {"command": "k6"}}}`)

	files := []core.PlannedFile{
		{
			Path:     ".mcp.json",
			Content:  newContent,
			Mode:     core.MergeJSONByKeyMode,
			MergeKey: "mcpServers.k6",
		},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outcomes[0].Status != core.Updated {
		t.Errorf("expected Updated, got %v", outcomes[0].Status)
	}

	data, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Both the original "other" and our "k6" should be present.
	if !contains(content, `"other"`) {
		t.Error("existing 'other' key was removed")
	}

	if !contains(content, `"k6"`) {
		t.Error("new 'k6' key was not added")
	}
}

func TestApply_MergeJSON_MalformedExisting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}

	files := []core.PlannedFile{
		{
			Path:     ".mcp.json",
			Content:  []byte(`{"mcpServers": {"k6": {}}}`),
			Mode:     core.MergeJSONByKeyMode,
			MergeKey: "mcpServers.k6",
		},
	}

	_, err := core.Apply(files, root, core.ApplyOptions{})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestApply_DryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	files := []core.PlannedFile{
		{Path: "test.txt", Content: []byte("hello"), Mode: core.CreateOnlyMode},
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outcomes[0].Status != core.Skipped {
		t.Errorf("expected Skipped in dry-run, got %v", outcomes[0].Status)
	}

	// File should not exist.
	if _, err := os.Stat(filepath.Join(root, "test.txt")); err == nil {
		t.Error("file was created despite dry-run")
	}
}

func TestApply_Idempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := []byte(`{"mcpServers": {"k6": {"command": "k6"}}}`)

	files := []core.PlannedFile{
		{
			Path:     ".mcp.json",
			Content:  content,
			Mode:     core.MergeJSONByKeyMode,
			MergeKey: "mcpServers.k6",
		},
	}

	// Run twice.
	if _, err := core.Apply(files, root, core.ApplyOptions{}); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	outcomes, err := core.Apply(files, root, core.ApplyOptions{})
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}

	if outcomes[0].Status != core.Updated {
		t.Errorf("expected Updated on second run, got %v", outcomes[0].Status)
	}
}

func TestApply_NoTouchUserFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Plant user-owned files.
	agentsMD := []byte("# My project agents\n")
	readmeMD := []byte("# My project\n")

	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), agentsMD, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), readmeMD, 0o600); err != nil {
		t.Fatal(err)
	}

	// Run init with some files.
	files := []core.PlannedFile{
		{Path: ".claude/skills/k6-test/SKILL.md", Content: []byte("skill"), Mode: core.CreateOnlyMode},
	}

	if _, err := core.Apply(files, root, core.ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Verify user files are untouched.
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(agentsMD) {
		t.Error("AGENTS.md was modified")
	}

	data, err = os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(readmeMD) {
		t.Error("README.md was modified")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
