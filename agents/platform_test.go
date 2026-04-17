// Tests for the data-driven Platform API. The fake FileSystem inevitably
// touches os.FileMode / os.FileInfo / os.PathError for interface conformance,
// hence the package-wide forbidigo suppression.
//
//nolint:forbidigo // test-only FileSystem fake mirrors stdlib shapes
package agents_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/xk6-agent/agents"
)

const testWorkspace = "/workspace"

// fakeFS is a tiny in-memory FileSystem used to exercise Install without
// touching the real disk.
type fakeFS struct {
	cwd   string
	files map[string][]byte
	dirs  map[string]bool
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		cwd:   testWorkspace,
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (f *fakeFS) WorkingDir() (string, error)               { return f.cwd, nil }
func (f *fakeFS) MkdirAll(path string, _ os.FileMode) error { f.dirs[path] = true; return nil }
func (f *fakeFS) Mkdir(path string, _ os.FileMode) error    { f.dirs[path] = true; return nil }
func (f *fakeFS) RemoveAll(path string) error {
	delete(f.dirs, path)
	for p := range f.files {
		if p == path || strings.HasPrefix(p, path+string(filepath.Separator)) {
			delete(f.files, p)
		}
	}
	return nil
}

func (f *fakeFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.files[path] = data
	return nil
}

func (f *fakeFS) Stat(path string) (os.FileInfo, error) {
	if _, ok := f.files[path]; ok {
		return fakeInfo{name: filepath.Base(path), dir: false}, nil
	}
	if f.dirs[path] {
		return fakeInfo{name: filepath.Base(path), dir: true}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
}

type fakeInfo struct {
	name string
	dir  bool
}

func (i fakeInfo) Name() string       { return i.name }
func (i fakeInfo) Size() int64        { return 0 }
func (i fakeInfo) Mode() os.FileMode  { return 0 }
func (i fakeInfo) IsDir() bool        { return i.dir }
func (i fakeInfo) ModTime() time.Time { return time.Time{} }
func (i fakeInfo) Sys() any           { return nil }

func TestPlatforms_HaveUniqueIDs(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, p := range agents.Platforms {
		if seen[p.ID] {
			t.Errorf("duplicate platform ID %q", p.ID)
		}
		seen[p.ID] = true
	}
	if len(seen) == 0 {
		t.Fatal("Platforms is empty")
	}
}

func TestPlatforms_Files_ProduceExpectedLayout(t *testing.T) {
	t.Parallel()

	type want struct {
		files []string // relative paths that must be present
	}
	cases := map[string]want{
		"claude": {
			files: []string{
				".claude/settings.local.json",
				".claude/agents/k6-test-generator.md",
				".claude/agents/k6-playwright-test-converter.md",
			},
		},
		"vscode": {
			files: []string{
				".vscode/mcp.json",
				".github/agents/k6-test-generator.agent.md",
				".github/agents/k6-playwright-test-converter.agent.md",
			},
		},
		"opencode": {
			files: []string{
				"opencode.json",
				".opencode/prompts/k6-test-generator.md",
				".opencode/prompts/k6-playwright-test-converter.md",
			},
		},
	}

	shared := agents.SharedAgents()
	for _, p := range agents.Platforms {
		w, ok := cases[p.ID]
		if !ok {
			t.Errorf("no test coverage for platform %q", p.ID)
			continue
		}
		files, err := p.Files(shared)
		if err != nil {
			t.Fatalf("%s: Files returned error: %v", p.ID, err)
		}
		got := make(map[string]bool, len(files))
		for _, f := range files {
			got[f.Path] = true
		}
		for _, want := range w.files {
			if !got[want] {
				t.Errorf("%s: missing expected file %q (got %v)", p.ID, want, keysOf(got))
			}
		}
	}
}

func TestInstall_WritesAllPlatformFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fs := newFakeFS()

	for _, p := range agents.Platforms {
		result, err := agents.Install(ctx, fs, p, agents.InitializerOptions{WorkingDir: "/workspace"})
		if err != nil {
			t.Fatalf("%s: Install returned %v", p.ID, err)
		}
		if len(result.FilesCreated) == 0 {
			t.Errorf("%s: Install produced no files", p.ID)
		}
		// Sanity: every file in the result actually landed in the fakeFS.
		for _, rel := range result.FilesCreated {
			abs := filepath.Join("/workspace", rel)
			if _, ok := fs.files[abs]; !ok {
				t.Errorf("%s: FilesCreated reports %q but nothing was written", p.ID, rel)
			}
		}
	}
}

func TestInstall_RejectsExistingRootWithoutForce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fs := newFakeFS()
	// Pretend .claude already exists.
	fs.dirs["/workspace/.claude"] = true

	_, err := agents.Install(ctx, fs, agents.Platforms[0], agents.InitializerOptions{
		WorkingDir: "/workspace",
	})
	if err == nil {
		t.Fatal("expected an error when root exists without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}
}

func TestInstall_ForceOverwritesExistingRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fs := newFakeFS()
	// Pretend .claude already exists with stale content.
	fs.dirs["/workspace/.claude"] = true
	fs.files["/workspace/.claude/stale.txt"] = []byte("remove me")

	_, err := agents.Install(ctx, fs, agents.Platforms[0], agents.InitializerOptions{
		WorkingDir: "/workspace",
		Force:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := fs.files["/workspace/.claude/stale.txt"]; ok {
		t.Error("--force should have removed stale content under .claude/")
	}
}

func TestInstall_DryRunWritesNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fs := newFakeFS()

	_, err := agents.Install(ctx, fs, agents.Platforms[0], agents.InitializerOptions{
		WorkingDir: "/workspace",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs.files) != 0 {
		t.Errorf("DryRun should not write any files, got %d", len(fs.files))
	}
}

func TestInstall_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fs := newFakeFS()
	_, err := agents.Install(ctx, fs, agents.Platforms[0], agents.InitializerOptions{
		WorkingDir: "/workspace",
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
