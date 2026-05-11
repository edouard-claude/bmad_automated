package cli_test

import (
	"context"
	"fmt"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/cli"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// Example_app demonstrates creating an App with custom dependencies for testing.
//
// In production, use [cli.NewApp] which wires up real implementations.
// For testing, construct App directly with mock dependencies to avoid
// spawning real processes.
func Example_app() {
	// Create mock dependencies
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "Done"},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}

	// Create app with injected dependencies
	app := &cli.App{
		Config:   &config.Config{},
		Executor: mockExecutor,
		Printer:  output.NewPrinter(),
		// Runner, StatusReader, StatusWriter would be set for full tests
	}

	// App fields are accessible for wiring up command handlers
	fmt.Println("executor configured:", app.Executor != nil)
	fmt.Println("printer configured:", app.Printer != nil)
	// Output:
	// executor configured: true
	// printer configured: true
}

// Example_commands demonstrates the available CLI commands and their purposes.
//
// Each command is attached to the root command via [cli.NewRootCommand].
// Commands use the App's injected dependencies for execution.
func Example_commands() {
	// Commands available in bmad-automate:
	//
	// Lifecycle commands (run full story workflow):
	//   run <story-key>     - Execute full lifecycle from current status to done
	//   queue <key>...      - Process multiple stories sequentially
	//   epic <epic-id>      - Process all stories in an epic
	//
	// Individual workflow commands:
	//   create-story <key>  - Create story from backlog
	//   dev-story <key>     - Develop story (ready-for-dev or in-progress)
	//   code-review <key>   - Review code (review status)
	//   git-commit <key>    - Commit changes after review
	//
	// Direct execution:
	//   raw <prompt>        - Execute a raw prompt directly

	fmt.Println("Lifecycle: run, queue, epic")
	fmt.Println("Workflows: create-story, dev-story, code-review, git-commit")
	fmt.Println("Direct: raw")
	// Output:
	// Lifecycle: run, queue, epic
	// Workflows: create-story, dev-story, code-review, git-commit
	// Direct: raw
}

// Example_statusInterfaces demonstrates the StatusReader and StatusWriter
// interfaces used for sprint-status.yaml management.
func Example_statusInterfaces() {
	// StatusReader reads story status from sprint-status.yaml
	// StatusWriter updates story status in sprint-status.yaml

	// Status values determine which workflow runs next:
	fmt.Println("backlog →", status.StatusReadyForDev)
	fmt.Println("ready-for-dev →", status.StatusInProgress)
	fmt.Println("in-progress →", status.StatusReview)
	fmt.Println("review →", status.StatusDone)
	// Output:
	// backlog → ready-for-dev
	// ready-for-dev → in-progress
	// in-progress → review
	// review → done
}

// Example_executeResult demonstrates the ExecuteResult type for testable CLI execution.
func Example_executeResult() {
	// ExecuteResult enables testing without os.Exit()
	result := cli.ExecuteResult{
		ExitCode: 0,
		Err:      nil,
	}

	if result.ExitCode == 0 {
		fmt.Println("success")
	} else {
		fmt.Println("failed:", result.Err)
	}
	// Output:
	// success
}

// Example_workflowRunner demonstrates the WorkflowRunner interface.
func Example_workflowRunner() {
	// WorkflowRunner is the interface for executing workflows.
	// Production implementation: workflow.Runner

	// RunSingle executes a named workflow
	// RunRaw executes a raw prompt directly

	// Mock implementation for testing:
	mock := &mockRunner{}
	ctx := context.Background()

	// Run a named workflow
	exitCode := mock.RunSingle(ctx, "create-story", "7-1-define-schema")
	fmt.Println("workflow exit code:", exitCode)

	// Run a raw prompt
	exitCode = mock.RunRaw(ctx, "List all Go files")
	fmt.Println("raw exit code:", exitCode)
	// Output:
	// workflow exit code: 0
	// raw exit code: 0
}

// mockRunner implements WorkflowRunner for examples.
type mockRunner struct{}

func (m *mockRunner) RunSingle(ctx context.Context, workflowName, storyKey string) int {
	return 0
}

func (m *mockRunner) RunRaw(ctx context.Context, prompt string) int {
	return 0
}
