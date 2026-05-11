// Package cli provides the command-line interface for bmad-automate.
//
// The cli package implements Cobra-based commands for orchestrating
// automated development workflows. It uses dependency injection via the
// [App] struct to wire up all required services, enabling comprehensive
// testing without subprocess execution.
//
// Key types:
//   - [App] - Main application container with injected dependencies
//   - [WorkflowRunner] - Interface for executing named workflows or raw prompts
//   - [StatusReader] - Interface for reading story status from sprint-status.yaml
//   - [StatusWriter] - Interface for updating story status
//   - [ExecuteResult] - Result type returned by testable entry points
//
// Commands provided:
//   - run - Execute full story lifecycle from current status to done
//   - queue - Run lifecycle for multiple stories sequentially
//   - epic - Run all stories in an epic
//   - raw - Execute a raw prompt directly
//   - create-story, dev-story, code-review, git-commit - Individual workflow commands
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/status"
	"github.com/edouard-claude/bmad_automated/internal/workflow"
)

// WorkflowRunner is the interface for executing development workflows.
//
// Implementations must handle workflow execution, output formatting, and
// return appropriate exit codes. The production implementation is
// [workflow.Runner], which orchestrates Claude CLI subprocess execution.
//
// Methods:
//   - RunSingle executes a named workflow (e.g., "create-story", "dev-story")
//     with a story key for template expansion
//   - RunRaw executes a raw prompt directly without workflow lookup
type WorkflowRunner interface {
	// RunSingle executes the named workflow for the given story key.
	// Returns 0 on success, non-zero on failure.
	RunSingle(ctx context.Context, workflowName, storyKey string) int

	// RunRaw executes a raw prompt directly without workflow lookup.
	// Returns 0 on success, non-zero on failure.
	RunRaw(ctx context.Context, prompt string) int
}

// StatusReader is the interface for reading story status from sprint-status.yaml.
//
// The production implementation is [status.Reader], which parses the YAML
// file at _bmad-output/implementation-artifacts/sprint-status.yaml.
type StatusReader interface {
	// Read returns the complete sprint status containing all story statuses.
	// Returns an error if the file cannot be read or parsed.
	Read() (*status.SprintStatus, error)

	// GetStoryStatus returns the current status of the given story key.
	// Returns an error if the story key is not found or the file cannot be read.
	GetStoryStatus(storyKey string) (status.Status, error)

	// GetEpicStories returns all story keys belonging to the given epic ID.
	// Story keys are sorted numerically by story number for predictable execution order.
	GetEpicStories(epicID string) ([]string, error)
}

// StatusWriter is the interface for updating story status in sprint-status.yaml.
//
// The production implementation is [status.Writer], which updates the YAML
// file atomically using temp file + rename to prevent corruption.
type StatusWriter interface {
	// UpdateStatus updates the status of the given story key to newStatus.
	// Returns an error if the story key is not found or the file cannot be written.
	UpdateStatus(storyKey string, newStatus status.Status) error
}

// App is the main application container with dependency injection.
//
// All dependencies are injected via struct fields, enabling comprehensive
// testing by substituting mock implementations. The production constructor
// [NewApp] wires up real implementations; tests can construct App directly
// with mock dependencies.
//
// Fields:
//   - Config: Application configuration loaded from workflows.yaml and environment
//   - Executor: Claude CLI executor for subprocess management
//   - Printer: Terminal output formatter using Lipgloss styles
//   - Runner: Workflow execution engine
//   - StatusReader: Sprint status file reader
//   - StatusWriter: Sprint status file writer
type App struct {
	// Config holds application configuration including workflow definitions.
	Config *config.Config

	// Executor runs Claude CLI as a subprocess and streams JSON events.
	Executor claude.Executor

	// Printer formats and displays output to the terminal.
	Printer output.Printer

	// Runner executes named workflows or raw prompts.
	Runner WorkflowRunner

	// StatusReader reads story status from sprint-status.yaml.
	StatusReader StatusReader

	// StatusWriter updates story status in sprint-status.yaml.
	StatusWriter StatusWriter
}

// NewApp creates a new [App] with all production dependencies wired up.
//
// This constructor initializes:
//   - A [claude.Executor] configured from cfg.Claude settings
//   - A [workflow.Runner] for workflow execution
//   - A [status.Reader] and [status.Writer] for sprint status management
//   - An [output.Printer] for terminal output
//
// For testing, construct [App] directly with mock dependencies instead.
func NewApp(cfg *config.Config) *App {
	printer := output.NewPrinter()

	executor := claude.NewExecutor(claude.ExecutorConfig{
		BinaryPath:   cfg.Claude.BinaryPath,
		OutputFormat: cfg.Claude.OutputFormat,
		StderrHandler: func(line string) {
			// Print stderr to stderr
			os.Stderr.WriteString("[stderr] " + line + "\n")
		},
	})

	runner := workflow.NewRunner(executor, printer, cfg)
	statusReader := status.NewReader("")
	statusWriter := status.NewWriter("")

	return &App{
		Config:       cfg,
		Executor:     executor,
		Printer:      printer,
		Runner:       runner,
		StatusReader: statusReader,
		StatusWriter: statusWriter,
	}
}

// NewRootCommand creates the root Cobra command with all subcommands attached.
//
// The command tree includes:
//   - run: Execute full story lifecycle from current status to done
//   - queue: Run lifecycle for multiple stories sequentially
//   - epic: Run all stories in an epic
//   - raw: Execute a raw prompt directly
//   - create-story: Create a new story from backlog status
//   - dev-story: Develop a story (ready-for-dev or in-progress status)
//   - code-review: Review code (review status)
//   - git-commit: Commit changes after review
func NewRootCommand(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bmad-automate",
		Short: "BMAD Automation CLI",
		Long: `BMAD Automation CLI - Automate development workflows with Claude.

This tool orchestrates Claude to run development workflows including
story creation, development, code review, and git operations.`,
	}

	// Add subcommands
	rootCmd.AddCommand(
		newCreateStoryCommand(app),
		newDevStoryCommand(app),
		newCodeReviewCommand(app),
		newGitCommitCommand(app),
		newRunCommand(app),
		newQueueCommand(app),
		newEpicCommand(app),
		newRawCommand(app),
		newStatusCommand(app),
	)

	return rootCmd
}

// ExecuteResult holds the result of running the CLI.
//
// This type enables testable CLI execution by returning exit codes and errors
// instead of calling os.Exit() directly. Use [Run] or [RunWithConfig] to get
// an ExecuteResult; [Execute] handles os.Exit() internally.
type ExecuteResult struct {
	// ExitCode is the exit code to return to the shell (0 = success).
	ExitCode int

	// Err is the error that caused a non-zero exit code, if any.
	Err error
}

// RunWithConfig creates the app and executes the root command with a pre-loaded config.
//
// This is the testable core of [Execute], accepting an already-loaded [config.Config]
// so tests can provide custom configurations. It creates an [App] via [NewApp],
// builds the command tree via [NewRootCommand], and executes the command.
//
// Exit codes:
//   - 0: Success
//   - 1: Config or command error
//   - Non-zero from subprocess: Passed through from Claude CLI
func RunWithConfig(cfg *config.Config) ExecuteResult {
	app := NewApp(cfg)
	rootCmd := NewRootCommand(app)

	if err := rootCmd.Execute(); err != nil {
		// Check if it's an ExitError from a command
		if code, ok := IsExitError(err); ok {
			return ExecuteResult{ExitCode: code, Err: err}
		}
		// Other errors (e.g., unknown command) - exit code 1
		return ExecuteResult{ExitCode: 1, Err: err}
	}
	return ExecuteResult{ExitCode: 0, Err: nil}
}

// Run loads configuration and executes the CLI, returning the result.
//
// This is the fully testable entry point that:
//  1. Loads configuration via [config.NewLoader]
//  2. Calls [RunWithConfig] with the loaded config
//
// Use this for integration tests that need to test config loading.
// For unit tests with custom configs, use [RunWithConfig] directly.
func Run() ExecuteResult {
	cfg, err := config.NewLoader().Load()
	if err != nil {
		return ExecuteResult{
			ExitCode: 1,
			Err:      fmt.Errorf("error loading config: %w", err),
		}
	}
	return RunWithConfig(cfg)
}

// Execute runs the CLI application and exits the process.
//
// This is the entry point called by main(). It calls [Run] and translates
// the [ExecuteResult] into an os.Exit() call. Because it exits the process,
// this function is not testable; use [Run] or [RunWithConfig] for tests.
func Execute() {
	result := Run()
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
}
