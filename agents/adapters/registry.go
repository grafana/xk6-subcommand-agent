package adapters

import "sort"

var registry = map[string]Target{} //nolint:gochecknoglobals // registry pattern requires package-level state

// Register adds a target to the global registry. It is called from
// each adapter's init() function.
func Register(t Target) {
	registry[t.Name()] = t
}

// Get returns a registered target by CLI name.
func Get(name string) (Target, bool) {
	t, ok := registry[name]
	return t, ok
}

// All returns every registered target, sorted by name.
func All() []Target {
	out := make([]Target, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})

	return out
}

// Names returns the sorted CLI names of all registered targets.
func Names() []string {
	targets := All()
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = t.Name()
	}

	return names
}
