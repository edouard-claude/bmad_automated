package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bmad-automate/internal/config"
	"bmad-automate/internal/status"
)

// TestQueueCommand_FullLifecycleExecution tests that queue command executes the full lifecycle for each story
func TestQueueCommand_FullLifecycleExecution(t *testing.T) {
	tests := []struct {
		name              string
		storyKeys         []string
		statusYAML        string
		expectedWorkflows []string
		expectedStatuses  []StatusUpdate
		expectError       bool
		failOnWorkflow    string
	}{
		{
			name:      "2 backlog stories runs full lifecycle for each",
			storyKeys: []string{"STORY-1", "STORY-2"},
			statusYAML: `development_status:
  STORY-1: backlog
  STORY-2: backlog`,
			// Each backlog story should run 4 workflows
			expectedWorkflows: []string{
				// Story STORY-1
				"create-story", "dev-story", "code-review", "git-commit",
				// Story STORY-2
				"create-story", "dev-story", "code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story STORY-1 lifecycle
				{StoryKey: "STORY-1", NewStatus: status.StatusReadyForDev},
				{StoryKey: "STORY-1", NewStatus: status.StatusReview},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				// Story STORY-2 lifecycle
				{StoryKey: "STORY-2", NewStatus: status.StatusReadyForDev},
				{StoryKey: "STORY-2", NewStatus: status.StatusReview},
				{StoryKey: "STORY-2", NewStatus: status.StatusDone},
				{StoryKey: "STORY-2", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:      "mixed statuses runs appropriate remaining workflows",
			storyKeys: []string{"STORY-1", "STORY-2", "STORY-3"},
			statusYAML: `development_status:
  STORY-1: backlog
  STORY-2: ready-for-dev
  STORY-3: review`,
			expectedWorkflows: []string{
				// Story STORY-1 (backlog): 4 workflows
				"create-story", "dev-story", "code-review", "git-commit",
				// Story STORY-2 (ready-for-dev): 3 workflows
				"dev-story", "code-review", "git-commit",
				// Story STORY-3 (review): 2 workflows
				"code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story STORY-1 lifecycle
				{StoryKey: "STORY-1", NewStatus: status.StatusReadyForDev},
				{StoryKey: "STORY-1", NewStatus: status.StatusReview},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				// Story STORY-2 lifecycle
				{StoryKey: "STORY-2", NewStatus: status.StatusReview},
				{StoryKey: "STORY-2", NewStatus: status.StatusDone},
				{StoryKey: "STORY-2", NewStatus: status.StatusDone},
				// Story STORY-3 lifecycle
				{StoryKey: "STORY-3", NewStatus: status.StatusDone},
				{StoryKey: "STORY-3", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:      "done story is skipped and runs others",
			storyKeys: []string{"STORY-1", "STORY-DONE", "STORY-3"},
			statusYAML: `development_status:
  STORY-1: backlog
  STORY-DONE: done
  STORY-3: ready-for-dev`,
			// Done story is skipped, others run full lifecycle
			expectedWorkflows: []string{
				// Story STORY-1 (backlog): 4 workflows
				"create-story", "dev-story", "code-review", "git-commit",
				// Story STORY-DONE (done): skipped, no workflows
				// Story STORY-3 (ready-for-dev): 3 workflows
				"dev-story", "code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story STORY-1 lifecycle
				{StoryKey: "STORY-1", NewStatus: status.StatusReadyForDev},
				{StoryKey: "STORY-1", NewStatus: status.StatusReview},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				// Story STORY-DONE (done): no status updates
				// Story STORY-3 lifecycle
				{StoryKey: "STORY-3", NewStatus: status.StatusReview},
				{StoryKey: "STORY-3", NewStatus: status.StatusDone},
				{StoryKey: "STORY-3", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:      "workflow failure mid-lifecycle stops processing",
			storyKeys: []string{"STORY-1", "STORY-2"},
			statusYAML: `development_status:
  STORY-1: backlog
  STORY-2: backlog`,
			failOnWorkflow: "dev-story",
			// First story: create-story succeeds, dev-story fails, stops
			expectedWorkflows: []string{"create-story", "dev-story"},
			expectedStatuses: []StatusUpdate{
				{StoryKey: "STORY-1", NewStatus: status.StatusReadyForDev},
			},
			expectError: true,
		},
		{
			name:      "all done stories returns success with no workflows",
			storyKeys: []string{"STORY-1", "STORY-2"},
			statusYAML: `development_status:
  STORY-1: done
  STORY-2: done`,
			expectedWorkflows: nil,
			expectedStatuses:  nil,
			expectError:       false,
		},
		{
			name:      "single story runs full lifecycle",
			storyKeys: []string{"STORY-1"},
			statusYAML: `development_status:
  STORY-1: backlog`,
			expectedWorkflows: []string{
				"create-story", "dev-story", "code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				{StoryKey: "STORY-1", NewStatus: status.StatusReadyForDev},
				{StoryKey: "STORY-1", NewStatus: status.StatusReview},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
				{StoryKey: "STORY-1", NewStatus: status.StatusDone},
			},
			expectError: false,
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

			args := append([]string{"queue"}, tt.storyKeys...)
			rootCmd.SetArgs(args)

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

func TestQueueCommand_StoryNotFoundReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, `development_status:
  OTHER-STORY: backlog`)

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
	rootCmd.SetArgs([]string{"queue", "STORY-NOT-FOUND"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)

	// No workflows should have been executed
	assert.Empty(t, mockRunner.ExecutedWorkflows)
}

func TestQueueCommand_MissingSprintStatusFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create sprint-status.yaml

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
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"queue", "STORY-1"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)
}

func TestResolveQueueArgs_StatusName(t *testing.T) {
	mock := &mockStatusReader{
		sprintStatus: &status.SprintStatus{
			DevelopmentStatus: map[string]status.Status{
				"3-1-tool":   status.StatusBacklog,
				"3-2-fetch":  status.StatusBacklog,
				"4-1-router": status.StatusInProgress,
				"3-3-done":   status.StatusDone,
			},
		},
	}

	keys, err := resolveQueueArgs([]string{"backlog"}, mock)
	require.NoError(t, err)
	assert.Equal(t, []string{"3-1-tool", "3-2-fetch"}, keys)
}

func TestResolveQueueArgs_SortedByEpicThenStory(t *testing.T) {
	mock := &mockStatusReader{
		sprintStatus: &status.SprintStatus{
			DevelopmentStatus: map[string]status.Status{
				"4-2-b":  status.StatusBacklog,
				"3-1-a":  status.StatusBacklog,
				"4-1-c":  status.StatusBacklog,
				"3-10-d": status.StatusBacklog,
			},
		},
	}

	keys, err := resolveQueueArgs([]string{"backlog"}, mock)
	require.NoError(t, err)
	assert.Equal(t, []string{"3-1-a", "3-10-d", "4-1-c", "4-2-b"}, keys)
}

func TestResolveQueueArgs_NoMatch(t *testing.T) {
	mock := &mockStatusReader{
		sprintStatus: &status.SprintStatus{
			DevelopmentStatus: map[string]status.Status{
				"3-1-a": status.StatusDone,
			},
		},
	}

	_, err := resolveQueueArgs([]string{"backlog"}, mock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no stories found with status: backlog")
}

func TestResolveQueueArgs_MultipleArgsPassthrough(t *testing.T) {
	// Multiple args should be passed through as-is, even if one looks like a status
	keys, err := resolveQueueArgs([]string{"backlog", "3-1-a"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"backlog", "3-1-a"}, keys)
}

func TestResolveQueueArgs_NonStatusPassthrough(t *testing.T) {
	// A single arg that is not a valid status should be passed through
	keys, err := resolveQueueArgs([]string{"3-1-tool"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"3-1-tool"}, keys)
}

func TestResolveQueueArgs_ReadError(t *testing.T) {
	mock := &mockStatusReader{err: assert.AnError}

	_, err := resolveQueueArgs([]string{"backlog"}, mock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read status")
}

func TestQueueCommand_StatusFilter(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, `development_status:
  3-1-tool: backlog
  3-2-fetch: backlog
  4-1-router: in-progress`)

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
	rootCmd.SetArgs([]string{"queue", "backlog"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	// Should have executed workflows for both backlog stories (4 each)
	assert.Len(t, mockRunner.ExecutedWorkflows, 8)
}

// Note: Legacy tests removed - obsolete after lifecycle executor change.
// The queue command now executes full lifecycle (multiple workflows per story), not single workflow routing.
// See TestQueueCommand_FullLifecycleExecution for comprehensive lifecycle testing.
