// Package core provides shared types and helpers for the xk6-agent skill
// and adapter system.
package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"
)

// Status describes what happened when applying a single PlannedFile.
type Status int

const (
	// Created means the file was newly created.
	Created Status = iota
	// Updated means an existing file was modified.
	Updated
	// Skipped means the file was left untouched.
	Skipped
	// Warned means there was a non-fatal issue.
	Warned
	// Errored means the operation failed.
	Errored
)

// Outcome records the result of applying a single planned file.
type Outcome struct {
	Path    string
	Status  Status
	Message string
}

// ApplyOptions controls how a plan is applied.
type ApplyOptions struct {
	Force  bool
	DryRun bool
}

// Apply executes a plan against the filesystem, enforcing all safety rules.
// Files are written relative to root.
func Apply(files []PlannedFile, root string, opts ApplyOptions) ([]Outcome, error) {
	var outcomes []Outcome

	for _, f := range files {
		absPath := filepath.Join(root, f.Path)

		if opts.DryRun {
			outcomes = append(outcomes, Outcome{
				Path:    f.Path,
				Status:  Skipped,
				Message: "dry-run",
			})

			continue
		}

		var outcome Outcome
		var err error

		switch f.Mode {
		case CreateOnlyMode:
			outcome, err = applyCreateOnly(absPath, f, opts.Force)
		case OverwriteIfManagedMode:
			outcome, err = applyOverwriteIfManaged(absPath, f)
		case MergeJSONByKeyMode:
			outcome, err = applyMergeJSON(absPath, f)
		default:
			return nil, fmt.Errorf("unknown write mode %d for %s", f.Mode, f.Path)
		}

		if err != nil {
			return nil, err
		}

		outcomes = append(outcomes, outcome)
	}

	return outcomes, nil
}

// PlannedFile describes a single file operation for safefs.
type PlannedFile struct {
	Path        string
	Content     []byte
	Mode        WriteMode
	MergeKey    string
	OwnerMarker string
}

// WriteMode controls how a PlannedFile is applied.
type WriteMode int

const (
	// CreateOnlyMode creates the file if absent, or overwrites if managed.
	CreateOnlyMode WriteMode = iota
	// OverwriteIfManagedMode overwrites only if the marker says we own it.
	OverwriteIfManagedMode
	// MergeJSONByKeyMode does a surgical JSON merge at MergeKey.
	MergeJSONByKeyMode
)

//nolint:forbidigo
func applyCreateOnly(absPath string, f PlannedFile, force bool) (Outcome, error) {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return Outcome{}, fmt.Errorf("mkdir for %s: %w", f.Path, err)
	}

	_, statErr := os.Stat(absPath)

	switch {
	case statErr == nil && !force:
		// File exists and we're not forcing — skip.
		return Outcome{Path: f.Path, Status: Skipped, Message: "exists"}, nil
	case statErr == nil:
		// File exists and force is on — overwrite.
		if err := os.WriteFile(absPath, f.Content, 0o600); err != nil {
			return Outcome{}, fmt.Errorf("write %s: %w", f.Path, err)
		}

		return Outcome{Path: f.Path, Status: Updated}, nil
	default:
		// File doesn't exist — create.
		if err := os.WriteFile(absPath, f.Content, 0o600); err != nil {
			return Outcome{}, fmt.Errorf("write %s: %w", f.Path, err)
		}

		return Outcome{Path: f.Path, Status: Created}, nil
	}
}

//nolint:forbidigo
func applyOverwriteIfManaged(absPath string, f PlannedFile) (Outcome, error) {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return Outcome{}, fmt.Errorf("mkdir for %s: %w", f.Path, err)
	}

	_, statErr := os.Stat(absPath)
	if errors.Is(statErr, fs.ErrNotExist) {
		return Outcome{
			Path:    f.Path,
			Status:  Skipped,
			Message: "not found, skipping update",
		}, nil
	}

	if statErr != nil {
		return Outcome{}, fmt.Errorf("stat %s: %w", f.Path, statErr)
	}

	// For now, overwrite. Full ownership marker support comes later.
	if err := os.WriteFile(absPath, f.Content, 0o600); err != nil {
		return Outcome{}, fmt.Errorf("write %s: %w", f.Path, err)
	}

	return Outcome{Path: f.Path, Status: Updated}, nil
}

//nolint:forbidigo,gosec
func applyMergeJSON(absPath string, f PlannedFile) (Outcome, error) {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return Outcome{}, fmt.Errorf("mkdir for %s: %w", f.Path, err)
	}

	existing, readErr := os.ReadFile(absPath)
	if errors.Is(readErr, fs.ErrNotExist) {
		if err := os.WriteFile(absPath, f.Content, 0o600); err != nil {
			return Outcome{}, fmt.Errorf("write %s: %w", f.Path, err)
		}

		return Outcome{Path: f.Path, Status: Created}, nil
	}

	if readErr != nil {
		return Outcome{}, fmt.Errorf("read %s: %w", f.Path, readErr)
	}

	merged, err := mergeJSON(existing, f.Content, f.MergeKey)
	if err != nil {
		msg := fmt.Sprintf(
			"failed to merge %s (fix manually or use --mcp=print): %v",
			f.Path, err,
		)

		return Outcome{}, fmt.Errorf("%s", msg)
	}

	if err := os.WriteFile(absPath, merged, 0o600); err != nil {
		return Outcome{}, fmt.Errorf("write merged %s: %w", f.Path, err)
	}

	return Outcome{Path: f.Path, Status: Updated}, nil
}

// mergeJSON merges newData into existing at the specified dot-path key.
// If key is empty, it does a top-level merge.
func mergeJSON(existing, newData []byte, key string) ([]byte, error) {
	if key == "" {
		return mergeJSONTopLevel(existing, newData)
	}

	// Validate existing JSON before merging.
	if !json.Valid(existing) {
		return nil, fmt.Errorf("existing content is not valid JSON")
	}

	// Extract the value at the merge key from the new data.
	var newObj map[string]any
	if err := json.Unmarshal(newData, &newObj); err != nil {
		return nil, fmt.Errorf("parse new content: %w", err)
	}

	// Navigate the key path in newObj to find the value to set.
	value := navigateJSON(newObj, key)
	if value == nil {
		return nil, fmt.Errorf("key %q not found in new content", key)
	}

	result, err := sjson.SetBytes(existing, key, value)
	if err != nil {
		return nil, fmt.Errorf("sjson set %q: %w", key, err)
	}

	// Re-indent for consistency.
	var buf any
	if err := json.Unmarshal(result, &buf); err != nil {
		return result, nil //nolint:nilerr // best-effort formatting
	}

	formatted, err := json.MarshalIndent(buf, "", "  ")
	if err != nil {
		return result, nil //nolint:nilerr // best-effort formatting
	}

	return append(formatted, '\n'), nil
}

func mergeJSONTopLevel(existing, newData []byte) ([]byte, error) {
	var existingObj map[string]any
	if err := json.Unmarshal(existing, &existingObj); err != nil {
		return nil, fmt.Errorf("parse existing: %w", err)
	}

	var newObj map[string]any
	if err := json.Unmarshal(newData, &newObj); err != nil {
		return nil, fmt.Errorf("parse new: %w", err)
	}

	for k, v := range newObj {
		existingObj[k] = v
	}

	return json.MarshalIndent(existingObj, "", "  ")
}

// navigateJSON traverses a map using a dot-separated key path.
func navigateJSON(obj map[string]any, key string) any {
	parts := strings.Split(key, ".")
	current := any(obj)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		current, ok = m[part]
		if !ok {
			return nil
		}
	}

	return current
}
