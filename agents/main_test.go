package agents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/xk6-agent/agents"
)

func TestInitializeAgents_AllSuccess(t *testing.T) {
	t.Parallel()
	claudeInit := &mockInitializer{name: "claude"}
	vscodeInit := &mockInitializer{name: "vscode"}

	err := agents.InitializeAgents(claudeInit, vscodeInit)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !claudeInit.initCalled || !vscodeInit.initCalled {
		t.Error("expected all initializers to be called")
	}
	if !claudeInit.validateCalled || !vscodeInit.validateCalled {
		t.Error("expected all validators to be called")
	}
}

func TestInitializeAgents_PartialFailure(t *testing.T) {
	t.Parallel()
	successInit := &mockInitializer{name: "success-agent"}
	failingInit := &mockInitializer{name: "failing-agent", initErr: errors.New("init failed")}
	subsequentInit := &mockInitializer{name: "subsequent-agent"}

	err := agents.InitializeAgents(successInit, failingInit, subsequentInit)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify error contains context about failing agent
	var initErr *agents.InitializationError
	if !errors.As(err, &initErr) {
		t.Error("expected InitializationError")
	}

	// All should be called despite error (accumulate strategy)
	if !successInit.initCalled || !failingInit.initCalled || !subsequentInit.initCalled {
		t.Error("expected all initializers to be called")
	}
}

func TestInitializeAgents_ValidationFailure(t *testing.T) {
	t.Parallel()
	validInit := &mockInitializer{name: "valid-agent"}
	invalidInit := &mockInitializer{name: "invalid-agent", validateErr: errors.New("validation failed")}
	anotherValidInit := &mockInitializer{name: "another-valid-agent"}

	err := agents.InitializeAgents(validInit, invalidInit, anotherValidInit)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify error contains context
	var initErr *agents.InitializationError
	if !errors.As(err, &initErr) {
		t.Error("expected InitializationError")
	}

	// Invalid agent should not be initialized due to validation failure
	if invalidInit.initCalled {
		t.Error("invalid agent should not be initialized after validation failure")
	}

	// Valid agents should still be initialized
	if !validInit.initCalled || !anotherValidInit.initCalled {
		t.Error("expected other agents to be initialized")
	}
}

func TestInitializeAgentsWithStrategy_FailFast(t *testing.T) {
	t.Parallel()
	firstInit := &mockInitializer{name: "first-agent"}
	failingInit := &mockInitializer{name: "failing-agent", initErr: errors.New("init failed")}
	skippedInit := &mockInitializer{name: "skipped-agent"}

	ctx := context.Background()
	opts := agents.InitializerOptions{}
	err := agents.InitializeAgentsWithStrategy(ctx, agents.StrategyFailFast, opts, firstInit, failingInit, skippedInit)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// With fail-fast, skipped agent should not be called
	if skippedInit.initCalled {
		t.Error("skipped agent should not be called with fail-fast strategy")
	}
}

func TestInitializeAgentsWithContext_Cancellation(t *testing.T) {
	t.Parallel()
	cancelledInit := &mockInitializer{name: "cancelled-agent"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := agents.InitializeAgentsWithContext(ctx, cancelledInit)

	if err == nil {
		t.Error("expected cancellation error, got nil")
	}

	// Validation should not be called due to cancellation
	if cancelledInit.validateCalled {
		t.Error("validation should not be called after context cancellation")
	}
}

type mockInitializer struct {
	name           string
	initErr        error
	validateErr    error
	initCalled     bool
	validateCalled bool
}

func (m *mockInitializer) Initialize(_ context.Context, _ agents.InitializerOptions) (*agents.InitializeResult, error) {
	m.initCalled = true
	if m.initErr != nil {
		return nil, m.initErr
	}
	return &agents.InitializeResult{
		FilesCreated: []string{"/test/file"},
	}, nil
}

func (m *mockInitializer) Name() string {
	return m.name
}

func (m *mockInitializer) Validate(_ context.Context, _ agents.InitializerOptions) error {
	m.validateCalled = true
	return m.validateErr
}
