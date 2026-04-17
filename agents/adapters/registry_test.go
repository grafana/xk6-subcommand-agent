package adapters_test

import (
	"context"
	"testing"

	"github.com/grafana/xk6-agent/agents/adapters"
)

type fakeTarget struct {
	name        string
	displayName string
}

func (f *fakeTarget) Name() string        { return f.name }
func (f *fakeTarget) DisplayName() string { return f.displayName }

func (f *fakeTarget) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{}
}

func (f *fakeTarget) Plan(_ context.Context, _ adapters.Inputs) (adapters.Plan, error) {
	return adapters.Plan{}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	adapters.Register(&fakeTarget{name: "test-target", displayName: "Test Target"})

	got, ok := adapters.Get("test-target")
	if !ok {
		t.Fatal("expected target to be registered")
	}

	if got.Name() != "test-target" {
		t.Errorf("expected name %q, got %q", "test-target", got.Name())
	}

	if got.DisplayName() != "Test Target" {
		t.Errorf("expected display name %q, got %q", "Test Target", got.DisplayName())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	t.Parallel()

	_, ok := adapters.Get("nonexistent-target")
	if ok {
		t.Fatal("expected target to not be found")
	}
}

func TestRegistry_All(t *testing.T) {
	t.Parallel()

	// Register targets in reverse order to test sorting.
	adapters.Register(&fakeTarget{name: "z-target", displayName: "Z"})
	adapters.Register(&fakeTarget{name: "a-target", displayName: "A"})

	all := adapters.All()
	if len(all) < 2 {
		t.Fatalf("expected at least 2 targets, got %d", len(all))
	}

	// Verify sorted order.
	for i := 1; i < len(all); i++ {
		if all[i-1].Name() >= all[i].Name() {
			t.Errorf("targets not sorted: %q >= %q", all[i-1].Name(), all[i].Name())
		}
	}
}

func TestRegistry_Names(t *testing.T) {
	t.Parallel()

	adapters.Register(&fakeTarget{name: "names-test", displayName: "Names Test"})

	names := adapters.Names()
	found := false
	for _, n := range names {
		if n == "names-test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'names-test' in Names()")
	}
}
