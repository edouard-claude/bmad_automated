package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

// mockStatusReader implements StatusReader for testing.
type mockStatusReader struct {
	sprintStatus *status.SprintStatus
	err          error
}

func (m *mockStatusReader) Read() (*status.SprintStatus, error) {
	return m.sprintStatus, m.err
}

func (m *mockStatusReader) GetStoryStatus(storyKey string) (status.Status, error) {
	if m.err != nil {
		return "", m.err
	}
	s, ok := m.sprintStatus.DevelopmentStatus[storyKey]
	if !ok {
		return "", nil
	}
	return s, nil
}

func (m *mockStatusReader) GetEpicStories(epicID string) ([]string, error) {
	return nil, nil
}

func newMockStatusReader(stories map[string]status.Status) *mockStatusReader {
	return &mockStatusReader{
		sprintStatus: &status.SprintStatus{
			DevelopmentStatus: stories,
		},
	}
}

func TestStatusCommand_BasicStructure(t *testing.T) {
	app := setupTestApp()
	cmd := newStatusCommand(app)

	assert.Equal(t, "status [epic-id]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Accepts 0 args
	err := cmd.Args(cmd, []string{})
	assert.NoError(t, err)

	// Accepts 1 arg
	err = cmd.Args(cmd, []string{"3"})
	assert.NoError(t, err)

	// Rejects 2 args
	err = cmd.Args(cmd, []string{"3", "4"})
	assert.Error(t, err)
}

func TestStatusCommand_HasAllFlag(t *testing.T) {
	app := setupTestApp()
	cmd := newStatusCommand(app)

	flag := cmd.Flags().Lookup("all")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestStatusCommand_DisplaysStories(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-tool-interface":  status.StatusDone,
		"3-2-read-file-tool":  status.StatusInProgress,
		"3-3-web-fetch-tool":  status.StatusBacklog,
		"4-1-message-router":  status.StatusBacklog,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	// Without --all, done stories should be hidden
	assert.NotContains(t, out, "3-1-tool-interface")
	// Active stories should be visible
	assert.Contains(t, out, "3-2-read-file-tool")
	assert.Contains(t, out, "3-3-web-fetch-tool")
	assert.Contains(t, out, "4-1-message-router")
	// Epic headers should be present
	assert.Contains(t, out, "Epic 3")
	assert.Contains(t, out, "Epic 4")
	// Summary should be present
	assert.Contains(t, out, "Summary:")
}

func TestStatusCommand_AllFlag(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-tool-interface":  status.StatusDone,
		"3-2-read-file-tool":  status.StatusInProgress,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status", "--all"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	// With --all, done stories should be visible
	assert.Contains(t, out, "3-1-tool-interface")
	assert.Contains(t, out, "3-2-read-file-tool")
	assert.Contains(t, out, "done")
}

func TestStatusCommand_EpicFilter(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-tool-interface":  status.StatusInProgress,
		"3-2-read-file-tool":  status.StatusBacklog,
		"4-1-message-router":  status.StatusBacklog,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status", "3"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	// Only epic 3 should be shown
	assert.Contains(t, out, "Epic 3")
	assert.Contains(t, out, "3-1-tool-interface")
	assert.NotContains(t, out, "Epic 4")
	assert.NotContains(t, out, "4-1-message-router")
}

func TestStatusCommand_ReadError(t *testing.T) {
	mock := &mockStatusReader{
		err: assert.AnError,
	}

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestStatusCommand_Summary(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-a": status.StatusDone,
		"3-2-b": status.StatusDone,
		"3-3-c": status.StatusInProgress,
		"4-1-d": status.StatusBacklog,
		"4-2-e": status.StatusReview,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status", "--all"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "1 in-progress")
	assert.Contains(t, out, "1 review")
	assert.Contains(t, out, "1 backlog")
	assert.Contains(t, out, "2 done")
}

func TestGroupByEpic(t *testing.T) {
	devStatus := map[string]status.Status{
		"3-1-a": status.StatusDone,
		"3-2-b": status.StatusInProgress,
		"4-1-c": status.StatusBacklog,
	}

	epics := groupByEpic(devStatus)

	// Should have 2 epics
	assert.Len(t, epics, 2)

	// Find epic 3
	var epic3 *epicGroup
	for i := range epics {
		if epics[i].id == 3 {
			epic3 = &epics[i]
		}
	}
	require.NotNil(t, epic3)
	assert.Len(t, epic3.stories, 2)
	// Should be sorted by story number
	assert.Equal(t, "3-1-a", epic3.stories[0].key)
	assert.Equal(t, "3-2-b", epic3.stories[1].key)
}

func TestGroupByEpic_NumericSorting(t *testing.T) {
	devStatus := map[string]status.Status{
		"3-10-last":   status.StatusBacklog,
		"3-2-middle":  status.StatusBacklog,
		"3-1-first":   status.StatusBacklog,
	}

	epics := groupByEpic(devStatus)
	require.Len(t, epics, 1)

	stories := epics[0].stories
	assert.Equal(t, "3-1-first", stories[0].key)
	assert.Equal(t, "3-2-middle", stories[1].key)
	assert.Equal(t, "3-10-last", stories[2].key)
}

func TestGroupByEpic_InvalidKeys(t *testing.T) {
	devStatus := map[string]status.Status{
		"invalid-key": status.StatusBacklog,
		"abc-1-test":  status.StatusBacklog,
		"3-abc-test":  status.StatusBacklog,
		"3-1-valid":   status.StatusDone,
	}

	epics := groupByEpic(devStatus)
	// Only the valid key should be included
	assert.Len(t, epics, 1)
	assert.Equal(t, 3, epics[0].id)
	assert.Len(t, epics[0].stories, 1)
}

func TestComputeEpicStatus(t *testing.T) {
	tests := []struct {
		name     string
		stories  []storyEntry
		expected string
	}{
		{
			name: "all done",
			stories: []storyEntry{
				{status: status.StatusDone},
				{status: status.StatusDone},
			},
			expected: "done",
		},
		{
			name: "has in-progress",
			stories: []storyEntry{
				{status: status.StatusDone},
				{status: status.StatusInProgress},
				{status: status.StatusBacklog},
			},
			expected: "in-progress",
		},
		{
			name: "has review",
			stories: []storyEntry{
				{status: status.StatusDone},
				{status: status.StatusReview},
			},
			expected: "in-progress",
		},
		{
			name: "all backlog",
			stories: []storyEntry{
				{status: status.StatusBacklog},
				{status: status.StatusBacklog},
			},
			expected: "backlog",
		},
		{
			name: "mixed with ready-for-dev",
			stories: []storyEntry{
				{status: status.StatusReadyForDev},
				{status: status.StatusBacklog},
			},
			expected: "backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeEpicStatus(tt.stories)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	counts := map[status.Status]int{
		status.StatusInProgress: 2,
		status.StatusBacklog:    3,
		status.StatusDone:       5,
	}

	renderSummary(buf, counts)

	out := buf.String()
	assert.Contains(t, out, "2 in-progress")
	assert.Contains(t, out, "3 backlog")
	assert.Contains(t, out, "5 done")
	// Order should be: in-progress before backlog before done
	ipIdx := strings.Index(out, "in-progress")
	blIdx := strings.Index(out, "backlog")
	dIdx := strings.Index(out, "done")
	assert.Less(t, ipIdx, blIdx)
	assert.Less(t, blIdx, dIdx)
}

func TestRenderStatus_StoriesAligned(t *testing.T) {
	buf := &bytes.Buffer{}
	devStatus := map[string]status.Status{
		"3-1-short":          status.StatusInProgress,
		"3-2-much-longer-name": status.StatusBacklog,
	}

	renderStatus(buf, devStatus, "", false)

	out := buf.String()
	lines := strings.Split(out, "\n")

	// Find the two story lines
	var storyLines []string
	for _, line := range lines {
		if strings.Contains(line, "3-1-") || strings.Contains(line, "3-2-") {
			storyLines = append(storyLines, line)
		}
	}
	require.Len(t, storyLines, 2)

	// Both status labels should be at the same column position
	idx1 := strings.Index(storyLines[0], "in-progress")
	idx2 := strings.Index(storyLines[1], "backlog")
	assert.Equal(t, idx1, idx2, "status labels should be aligned")
}

func TestStatusCommand_EmptyStatus(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Summary:")
}

func TestStatusCommand_AllDone_NoAll(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-a": status.StatusDone,
		"3-2-b": status.StatusDone,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	// No epic headers since all are done and --all not set
	assert.NotContains(t, out, "Epic 3")
	// But summary should still count them
	assert.Contains(t, out, "2 done")
}

func TestStatusCommand_EpicFilter_NoMatch(t *testing.T) {
	mock := newMockStatusReader(map[string]status.Status{
		"3-1-a": status.StatusInProgress,
	})

	app := &App{StatusReader: mock}
	rootCmd := NewRootCommand(app)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"status", "99"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.NotContains(t, out, "Epic")
	assert.Contains(t, out, "Summary:")
}
