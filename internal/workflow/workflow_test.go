package workflow

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
)

func setupTestRunner() (*Runner, *claude.MockExecutor, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	cfg := config.DefaultConfig()
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "Working on it..."},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}
	runner := NewRunner(mockExecutor, printer, cfg)
	return runner, mockExecutor, buf
}

func TestNewRunner(t *testing.T) {
	cfg := config.DefaultConfig()
	printer := output.NewPrinter()
	executor := &claude.MockExecutor{}

	runner := NewRunner(executor, printer, cfg)

	assert.NotNil(t, runner)
}

func TestRunner_RunSingle(t *testing.T) {
	runner, mockExecutor, _ := setupTestRunner()

	ctx := context.Background()
	exitCode := runner.RunSingle(ctx, "create-story", "test-123")

	assert.Equal(t, 0, exitCode)
	require.Len(t, mockExecutor.RecordedPrompts, 1)
	assert.Contains(t, mockExecutor.RecordedPrompts[0], "test-123")
}

func TestRunner_RunSingle_UnknownWorkflow(t *testing.T) {
	runner, _, _ := setupTestRunner()

	ctx := context.Background()
	exitCode := runner.RunSingle(ctx, "unknown-workflow", "test-123")

	assert.Equal(t, 1, exitCode)
}

func TestRunner_RunRaw(t *testing.T) {
	runner, mockExecutor, _ := setupTestRunner()

	ctx := context.Background()
	exitCode := runner.RunRaw(ctx, "custom prompt")

	assert.Equal(t, 0, exitCode)
	require.Len(t, mockExecutor.RecordedPrompts, 1)
	assert.Equal(t, "custom prompt", mockExecutor.RecordedPrompts[0])
}

func TestRunner_RunFullCycle_Success(t *testing.T) {
	runner, mockExecutor, _ := setupTestRunner()

	ctx := context.Background()
	exitCode := runner.RunFullCycle(ctx, "test-story")

	assert.Equal(t, 0, exitCode)
	// Should have 4 prompts (one for each step in full cycle)
	assert.Len(t, mockExecutor.RecordedPrompts, 4)
}

func TestRunner_RunFullCycle_FailAtStep(t *testing.T) {
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	cfg := config.DefaultConfig()

	callCount := 0
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
	}

	// Make it fail on second call
	originalExecute := mockExecutor.ExecuteWithResult
	_ = originalExecute

	runner := NewRunner(mockExecutor, printer, cfg)

	// Override executor to fail on second call
	failingExecutor := &failOnNthCallExecutor{
		inner:   mockExecutor,
		failOn:  2,
		current: &callCount,
	}
	runner.executor = failingExecutor

	ctx := context.Background()
	exitCode := runner.RunFullCycle(ctx, "test-story")

	assert.Equal(t, 1, exitCode)
}

// failOnNthCallExecutor wraps an executor and fails on the nth call
type failOnNthCallExecutor struct {
	inner   *claude.MockExecutor
	failOn  int
	current *int
}

func (f *failOnNthCallExecutor) Execute(ctx context.Context, prompt string) (<-chan claude.Event, error) {
	return f.inner.Execute(ctx, prompt)
}

func (f *failOnNthCallExecutor) ExecuteWithResult(ctx context.Context, prompt string, handler claude.EventHandler) (int, error) {
	*f.current++
	if *f.current == f.failOn {
		return 1, nil
	}
	return f.inner.ExecuteWithResult(ctx, prompt, handler)
}

func TestRunner_HandleEvent(t *testing.T) {
	runner, _, buf := setupTestRunner()

	// Test session start
	runner.handleEvent(claude.Event{Type: claude.EventTypeSystem, SessionStarted: true})
	assert.Contains(t, buf.String(), "Session started")

	buf.Reset()

	// Test text
	runner.handleEvent(claude.Event{Type: claude.EventTypeAssistant, Text: "Hello!"})
	assert.Contains(t, buf.String(), "Hello!")

	buf.Reset()

	// Test tool use
	runner.handleEvent(claude.Event{
		Type:            claude.EventTypeAssistant,
		ToolName:        "Bash",
		ToolCommand:     "ls",
		ToolDescription: "List files",
	})
	assert.Contains(t, buf.String(), "Bash")

	buf.Reset()

	// Test tool result
	runner.handleEvent(claude.Event{
		Type:       claude.EventTypeUser,
		ToolStdout: "file1.go",
	})
	assert.Contains(t, buf.String(), "file1.go")
}

func TestStepResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		expected bool
	}{
		{"zero exit code", 0, true},
		{"non-zero exit code", 1, false},
		{"another non-zero", 127, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StepResult{ExitCode: tt.exitCode}
			assert.Equal(t, tt.expected, result.IsSuccess())
		})
	}
}

// QueueRunner tests

func TestNewQueueRunner(t *testing.T) {
	runner, _, _ := setupTestRunner()
	queueRunner := NewQueueRunner(runner)

	assert.NotNil(t, queueRunner)
	assert.Equal(t, runner, queueRunner.runner)
}

// Note: QueueRunner.RunQueueWithStatus tests are in internal/cli/queue_test.go
// since they require status.Reader and full CLI integration testing
