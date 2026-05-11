package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// TestEpicCommand_FullLifecycleExecution tests that epic command executes the full lifecycle for each story
func TestEpicCommand_FullLifecycleExecution(t *testing.T) {
	tests := []struct {
		name              string
		epicID            string
		statusYAML        string
		expectedWorkflows []string
		expectedStatuses  []StatusUpdate
		expectError       bool
		failOnWorkflow    string
	}{
		{
			name:   "epic with 2 backlog stories runs full lifecycle for each",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: backlog`,
			// Each backlog story should run 4 workflows
			expectedWorkflows: []string{
				// Story 6-1
				"create-story", "dev-story", "code-review", "git-commit",
				// Story 6-2
				"create-story", "dev-story", "code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1 lifecycle
				{StoryKey: "6-1-first", NewStatus: status.StatusReadyForDev},
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-2 lifecycle
				{StoryKey: "6-2-second", NewStatus: status.StatusReadyForDev},
				{StoryKey: "6-2-second", NewStatus: status.StatusReview},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "epic with mixed statuses runs appropriate remaining workflows",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: ready-for-dev
  6-3-third: review`,
			expectedWorkflows: []string{
				// Story 6-1 (backlog): 4 workflows
				"create-story", "dev-story", "code-review", "git-commit",
				// Story 6-2 (ready-for-dev): 3 workflows
				"dev-story", "code-review", "git-commit",
				// Story 6-3 (review): 2 workflows
				"code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1 lifecycle
				{StoryKey: "6-1-first", NewStatus: status.StatusReadyForDev},
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-2 lifecycle
				{StoryKey: "6-2-second", NewStatus: status.StatusReview},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
				// Story 6-3 lifecycle
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "epic with done story skips done and runs others",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-done: done
  6-3-third: ready-for-dev`,
			// Done story is skipped, others run full lifecycle
			expectedWorkflows: []string{
				// Story 6-1 (backlog): 4 workflows
				"create-story", "dev-story", "code-review", "git-commit",
				// Story 6-2 (done): skipped, no workflows
				// Story 6-3 (ready-for-dev): 3 workflows
				"dev-story", "code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1 lifecycle
				{StoryKey: "6-1-first", NewStatus: status.StatusReadyForDev},
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-2 (done): no status updates
				// Story 6-3 lifecycle
				{StoryKey: "6-3-third", NewStatus: status.StatusReview},
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "story failure mid-lifecycle stops processing",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: backlog`,
			failOnWorkflow: "dev-story",
			// First story: create-story succeeds, dev-story fails, stops
			expectedWorkflows: []string{"create-story", "dev-story"},
			expectedStatuses: []StatusUpdate{
				{StoryKey: "6-1-first", NewStatus: status.StatusReadyForDev},
			},
			expectError: true,
		},
		{
			name:   "all stories done returns success with no workflows",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: done
  6-2-second: done`,
			expectedWorkflows: nil,
			expectedStatuses:  nil,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createSprintStatusFile(t, tmpDir, tt.statusYAML)

			mockRunner := &MockWorkflowRunner{
				FailOnWorkflow: tt.failOnWorkflow,
			}
			mockWriter := &MockStatusWriter{}
			statusReader := status.NewReader(tmpDir)

			app := &App{
				Config:       config.DefaultConfig(),
				StatusReader: statusReader,
				StatusWriter: mockWriter,
				Runner:       mockRunner,
			}

			rootCmd := NewRootCommand(app)
			outBuf := &bytes.Buffer{}
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)
			rootCmd.SetArgs([]string{"epic", tt.epicID})

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
				"workflows should be executed in lifecycle order for each story")

			// Verify status updates occurred after each workflow
			if tt.expectedStatuses != nil {
				require.Len(t, mockWriter.Updates, len(tt.expectedStatuses),
					"should have correct number of status updates")
				for i, expected := range tt.expectedStatuses {
					assert.Equal(t, expected.StoryKey, mockWriter.Updates[i].StoryKey,
						"status update %d should be for story %s", i, expected.StoryKey)
					assert.Equal(t, expected.NewStatus, mockWriter.Updates[i].NewStatus,
						"status update %d should be %s", i, expected.NewStatus)
				}
			} else {
				assert.Empty(t, mockWriter.Updates, "should have no status updates")
			}
		})
	}
}

func TestEpicCommand_NoStoriesFoundReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, `development_status:
  7-1-other-epic: backlog`)

	mockRunner := &MockWorkflowRunner{}
	mockWriter := &MockStatusWriter{}
	statusReader := status.NewReader(tmpDir)

	app := &App{
		Config:       config.DefaultConfig(),
		StatusReader: statusReader,
		StatusWriter: mockWriter,
		Runner:       mockRunner,
	}

	rootCmd := NewRootCommand(app)
	outBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)
	rootCmd.SetArgs([]string{"epic", "6"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)

	// No workflows should have been executed
	assert.Empty(t, mockRunner.ExecutedWorkflows)
}

// Note: Legacy tests removed - obsolete after lifecycle executor change.
// The epic command now executes full lifecycle (multiple workflows per story), not single workflow routing.
// See TestEpicCommand_FullLifecycleExecution for comprehensive lifecycle testing.
