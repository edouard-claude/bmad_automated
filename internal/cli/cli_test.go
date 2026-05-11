package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/status"
	"github.com/edouard-claude/bmad_automated/internal/workflow"
)

func setupTestApp() *App {
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
	statusReader := status.NewReader("")

	return &App{
		Config:       cfg,
		Executor:     mockExecutor,
		Printer:      printer,
		Runner:       runner,
		StatusReader: statusReader,
	}
}

func TestNewApp(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	assert.NotNil(t, app)
	assert.NotNil(t, app.Config)
	assert.NotNil(t, app.Executor)
	assert.NotNil(t, app.Printer)
	assert.NotNil(t, app.Runner)
	assert.NotNil(t, app.StatusReader)
	assert.Equal(t, cfg, app.Config)
}

func TestNewRootCommand(t *testing.T) {
	app := setupTestApp()
	rootCmd := NewRootCommand(app)

	assert.NotNil(t, rootCmd)
	assert.Equal(t, "bmad-automate", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "BMAD")
}

func TestNewRootCommand_HasAllSubcommands(t *testing.T) {
	app := setupTestApp()
	rootCmd := NewRootCommand(app)

	expectedCommands := []string{
		"create-story",
		"dev-story",
		"code-review",
		"git-commit",
		"run",
		"queue",
		"raw",
		"status",
	}

	commands := rootCmd.Commands()
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	for _, expected := range expectedCommands {
		assert.Contains(t, commandNames, expected, "missing subcommand: %s", expected)
	}
}

func TestCreateStoryCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newCreateStoryCommand(app)

	assert.Equal(t, "create-story <story-key>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test args validation - should require exactly 1 arg
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"story-1", "extra"})
	assert.Error(t, err)
}

func TestDevStoryCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newDevStoryCommand(app)

	assert.Equal(t, "dev-story <story-key>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// Test args validation
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)
}

func TestCodeReviewCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newCodeReviewCommand(app)

	assert.Equal(t, "code-review <story-key>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// Test args validation
	err := cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)
}

func TestGitCommitCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newGitCommitCommand(app)

	assert.Equal(t, "git-commit <story-key>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// Test args validation
	err := cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)
}

func TestRunCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newRunCommand(app)

	assert.Equal(t, "run <story-key>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test args validation
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)
}

func TestQueueCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newQueueCommand(app)

	assert.Equal(t, "queue <story-key|status> [story-key...]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test args validation - should require at least 1 arg
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"story-1"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"story-1", "story-2", "story-3"})
	assert.NoError(t, err)
}

func TestRawCommand(t *testing.T) {
	app := setupTestApp()
	cmd := newRawCommand(app)

	assert.Equal(t, "raw <prompt>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test args validation - should require at least 1 arg
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"hello"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"hello", "world", "test"})
	assert.NoError(t, err)
}

func TestRootCommand_Help(t *testing.T) {
	app := setupTestApp()
	rootCmd := NewRootCommand(app)

	// Capture help output
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	helpOutput := buf.String()
	assert.Contains(t, helpOutput, "BMAD")
	assert.Contains(t, helpOutput, "Available Commands")
}

func TestSubcommand_Help(t *testing.T) {
	app := setupTestApp()
	rootCmd := NewRootCommand(app)

	tests := []struct {
		name    string
		command string
	}{
		{"create-story help", "create-story"},
		{"dev-story help", "dev-story"},
		{"code-review help", "code-review"},
		{"git-commit help", "git-commit"},
		{"run help", "run"},
		{"queue help", "queue"},
		{"raw help", "raw"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetArgs([]string{tt.command, "--help"})

			err := rootCmd.Execute()
			require.NoError(t, err)

			helpOutput := buf.String()
			assert.NotEmpty(t, helpOutput)
		})
	}
}

// findCommand finds a subcommand by name
func findCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func TestCommandsHaveRunEFunctions(t *testing.T) {
	app := setupTestApp()
	rootCmd := NewRootCommand(app)

	commands := []string{
		"create-story",
		"dev-story",
		"code-review",
		"git-commit",
		"run",
		"queue",
		"raw",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findCommand(rootCmd, cmdName)
			require.NotNil(t, cmd, "command %s not found", cmdName)
			assert.NotNil(t, cmd.RunE, "command %s should have a RunE function", cmdName)
		})
	}
}

// setupFailingTestApp creates an App with a mock executor that returns exit code 1.
func setupFailingTestApp() *App {
	cfg := config.DefaultConfig()
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 1, // Simulate failure
	}
	runner := workflow.NewRunner(mockExecutor, printer, cfg)
	statusReader := status.NewReader("")

	return &App{
		Config:       cfg,
		Executor:     mockExecutor,
		Printer:      printer,
		Runner:       runner,
		StatusReader: statusReader,
	}
}

func TestCommandExecution_Success(t *testing.T) {
	// Note: "run" and "queue" commands excluded - they require sprint-status.yaml and are tested in run_test.go/queue_test.go
	tests := []struct {
		command string
		args    []string
	}{
		{"create-story", []string{"TEST-123"}},
		{"dev-story", []string{"TEST-123"}},
		{"code-review", []string{"TEST-123"}},
		{"git-commit", []string{"TEST-123"}},
		{"raw", []string{"hello", "world"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			app := setupTestApp()
			rootCmd := NewRootCommand(app)

			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(append([]string{tt.command}, tt.args...))

			err := rootCmd.Execute()
			assert.NoError(t, err)
		})
	}
}

func TestCommandExecution_Failure(t *testing.T) {
	// Note: "run" and "queue" commands excluded - they require sprint-status.yaml and are tested in run_test.go/queue_test.go
	tests := []struct {
		command string
		args    []string
	}{
		{"create-story", []string{"TEST-123"}},
		{"dev-story", []string{"TEST-123"}},
		{"code-review", []string{"TEST-123"}},
		{"git-commit", []string{"TEST-123"}},
		{"raw", []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			app := setupFailingTestApp()
			rootCmd := NewRootCommand(app)

			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(append([]string{tt.command}, tt.args...))

			err := rootCmd.Execute()
			require.Error(t, err)

			code, ok := IsExitError(err)
			assert.True(t, ok, "error should be an ExitError")
			assert.Equal(t, 1, code)
		})
	}
}

func TestCommandExecution_InvalidArgs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{"create-story no args", "create-story", []string{}},
		{"dev-story no args", "dev-story", []string{}},
		{"code-review no args", "code-review", []string{}},
		{"git-commit no args", "git-commit", []string{}},
		{"run no args", "run", []string{}},
		{"queue no args", "queue", []string{}},
		{"raw no args", "raw", []string{}},
		{"create-story too many args", "create-story", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp()
			rootCmd := NewRootCommand(app)

			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(append([]string{tt.command}, tt.args...))

			err := rootCmd.Execute()
			require.Error(t, err)
		})
	}
}

func TestRunWithConfig_Success(t *testing.T) {
	cfg := config.DefaultConfig()

	// With no args, rootCmd.Execute() shows help and returns nil
	result := RunWithConfig(cfg)

	assert.Equal(t, 0, result.ExitCode)
	assert.NoError(t, result.Err)
}

func TestExecuteResult(t *testing.T) {
	t.Run("zero exit code", func(t *testing.T) {
		result := ExecuteResult{ExitCode: 0, Err: nil}
		assert.Equal(t, 0, result.ExitCode)
		assert.NoError(t, result.Err)
	})

	t.Run("non-zero exit code with error", func(t *testing.T) {
		result := ExecuteResult{ExitCode: 1, Err: NewExitError(1)}
		assert.Equal(t, 1, result.ExitCode)
		assert.Error(t, result.Err)
	})
}
