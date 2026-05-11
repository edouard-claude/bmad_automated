package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/status"
	"github.com/edouard-claude/bmad_automated/internal/workflow"
)

// MockWorkflowRunner records workflow executions for testing.
type MockWorkflowRunner struct {
	ExecutedWorkflows []string
	ReturnExitCode    int
	FailOnWorkflow    string // If set, fail when this workflow is called
}

func (m *MockWorkflowRunner) RunSingle(ctx context.Context, workflowName, storyKey string) int {
	m.ExecutedWorkflows = append(m.ExecutedWorkflows, workflowName)
	if m.FailOnWorkflow == workflowName {
		return 1
	}
	return m.ReturnExitCode
}

func (m *MockWorkflowRunner) RunRaw(ctx context.Context, prompt string) int {
	return m.ReturnExitCode
}

// MockStatusWriter records status updates for testing.
type MockStatusWriter struct {
	Updates        []StatusUpdate
	FailOnStoryKey string
}

type StatusUpdate struct {
	StoryKey  string
	NewStatus status.Status
}

func (m *MockStatusWriter) UpdateStatus(storyKey string, newStatus status.Status) error {
	m.Updates = append(m.Updates, StatusUpdate{StoryKey: storyKey, NewStatus: newStatus})
	if m.FailOnStoryKey == storyKey {
		return fmt.Errorf("story not found: %s", storyKey)
	}
	return nil
}

func setupRunTestApp(tmpDir string) (*App, *claude.MockExecutor, *bytes.Buffer) {
	cfg := config.DefaultConfig()
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}
	runner := workflow.NewRunner(mockExecutor, printer, cfg)
	statusReader := status.NewReader(tmpDir)
	statusWriter := status.NewWriter(tmpDir)

	return &App{
		Config:       cfg,
		Executor:     mockExecutor,
		Printer:      printer,
		Runner:       runner,
		StatusReader: statusReader,
		StatusWriter: statusWriter,
	}, mockExecutor, buf
}

func createSprintStatusFile(t *testing.T, tmpDir string, content string) {
	t.Helper()
	artifactsDir := filepath.Join(tmpDir, "_bmad-output", "implementation-artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(artifactsDir, "sprint-status.yaml"), []byte(content), 0644))
}

// Note: TestRunCommand_StatusBasedRouting removed - obsolete after lifecycle executor change.
// The run command now executes full lifecycle (multiple workflows), not single workflow routing.
// See TestRunCommand_FullLifecycleExecution for comprehensive lifecycle testing.

func TestRunCommand_MissingSprintStatusFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create sprint-status.yaml

	app, _, _ := setupRunTestApp(tmpDir)
	rootCmd := NewRootCommand(app)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"run", "STORY-1"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)
}

// Note: TestRunCommand_WorkflowExecution removed - obsolete after lifecycle executor change.
// The run command now executes full lifecycle (multiple workflows), not single workflow routing.
// See TestRunCommand_FullLifecycleExecution for comprehensive lifecycle testing.

// Note: TestRunCommand_WorkflowFailure removed - covered by TestRunCommand_FullLifecycleExecution
// "workflow failure mid-lifecycle returns error" test case.

// TestRunCommand_FullLifecycleExecution tests that run command executes the full lifecycle
func TestRunCommand_FullLifecycleExecution(t *testing.T) {
	tests := []struct {
		name              string
		storyKey          string
		initialStatus     string
		expectedWorkflows []string
		expectedStatuses  []status.Status
		expectError       bool
		failOnWorkflow    string
	}{
		{
			name:          "backlog story runs full lifecycle (4 workflows)",
			storyKey:      "STORY-1",
			initialStatus: "backlog",
			expectedWorkflows: []string{
				"create-story",
				"dev-story",
				"code-review",
				"git-commit",
			},
			expectedStatuses: []status.Status{
				status.StatusReadyForDev,
				status.StatusReview,
				status.StatusDone,
				status.StatusDone,
			},
			expectError: false,
		},
		{
			name:          "ready-for-dev story runs 3 workflows",
			storyKey:      "STORY-2",
			initialStatus: "ready-for-dev",
			expectedWorkflows: []string{
				"dev-story",
				"code-review",
				"git-commit",
			},
			expectedStatuses: []status.Status{
				status.StatusReview,
				status.StatusDone,
				status.StatusDone,
			},
			expectError: false,
		},
		{
			name:          "review story runs 2 workflows",
			storyKey:      "STORY-3",
			initialStatus: "review",
			expectedWorkflows: []string{
				"code-review",
				"git-commit",
			},
			expectedStatuses: []status.Status{
				status.StatusDone,
				status.StatusDone,
			},
			expectError: false,
		},
		{
			name:              "done story prints message and exits 0",
			storyKey:          "STORY-4",
			initialStatus:     "done",
			expectedWorkflows: nil, // No workflows executed
			expectedStatuses:  nil,
			expectError:       false,
		},
		{
			name:              "workflow failure mid-lifecycle returns error",
			storyKey:          "STORY-5",
			initialStatus:     "backlog",
			failOnWorkflow:    "dev-story",
			expectedWorkflows: []string{"create-story", "dev-story"},
			expectedStatuses:  []status.Status{status.StatusReadyForDev},
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createSprintStatusFile(t, tmpDir, "development_status:\n  "+tt.storyKey+": "+tt.initialStatus)

			mockRunner := &MockWorkflowRunner{
				FailOnWorkflow: tt.failOnWorkflow,
			}
			mockWriter := &MockStatusWriter{}
			statusReader := status.NewReader(tmpDir)
			printer := output.NewPrinterWithWriter(&bytes.Buffer{}) // Capture output

			app := &App{
				Config:       config.DefaultConfig(),
				StatusReader: statusReader,
				StatusWriter: mockWriter,
				Runner:       mockRunner,
				Printer:      printer,
			}

			rootCmd := NewRootCommand(app)
			outBuf := &bytes.Buffer{}
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)
			rootCmd.SetArgs([]string{"run", tt.storyKey})

			err := rootCmd.Execute()

			if tt.expectError {
				require.Error(t, err)
				code, ok := IsExitError(err)
				assert.True(t, ok, "error should be an ExitError")
				assert.Equal(t, 1, code)
			} else {
				assert.NoError(t, err)
			}

			// Verify workflows were executed in order
			assert.Equal(t, tt.expectedWorkflows, mockRunner.ExecutedWorkflows,
				"workflows should be executed in lifecycle order")

			// Verify status updates occurred after each workflow
			if tt.expectedStatuses != nil {
				require.Len(t, mockWriter.Updates, len(tt.expectedStatuses),
					"should have correct number of status updates")
				for i, expectedStatus := range tt.expectedStatuses {
					assert.Equal(t, tt.storyKey, mockWriter.Updates[i].StoryKey)
					assert.Equal(t, expectedStatus, mockWriter.Updates[i].NewStatus,
						"status update %d should be %s", i, expectedStatus)
				}
			}
		})
	}
}

func TestRunCommand_LifecycleStoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, "development_status:\n  OTHER-STORY: backlog")

	mockRunner := &MockWorkflowRunner{}
	mockWriter := &MockStatusWriter{}
	statusReader := status.NewReader(tmpDir)
	printer := output.NewPrinterWithWriter(&bytes.Buffer{}) // Capture output

	app := &App{
		Config:       config.DefaultConfig(),
		StatusReader: statusReader,
		StatusWriter: mockWriter,
		Runner:       mockRunner,
		Printer:      printer,
	}

	rootCmd := NewRootCommand(app)
	outBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)
	rootCmd.SetArgs([]string{"run", "STORY-NOT-FOUND"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)

	// No workflows should have been executed
	assert.Empty(t, mockRunner.ExecutedWorkflows)
}
