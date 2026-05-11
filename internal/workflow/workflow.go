package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
)

// Runner orchestrates workflow execution using Claude CLI.
//
// Runner is the primary executor for development workflows. It combines a
// [claude.Executor] for spawning Claude processes, an [output.Printer] for
// formatted terminal output, and a [config.Config] for prompt templates.
//
// Use [NewRunner] to create a properly initialized Runner instance.
type Runner struct {
	executor claude.Executor
	printer  output.Printer
	config   *config.Config
}

// NewRunner creates a new workflow runner with the specified dependencies.
//
// Parameters:
//   - executor: The [claude.Executor] implementation for running Claude CLI
//   - printer: The [output.Printer] for formatted terminal output
//   - cfg: The configuration containing workflow prompt templates
//
// The executor typically uses [claude.NewExecutor] in production or
// [claude.MockExecutor] for testing.
func NewRunner(executor claude.Executor, printer output.Printer, cfg *config.Config) *Runner {
	return &Runner{
		executor: executor,
		printer:  printer,
		config:   cfg,
	}
}

// RunSingle executes a single named workflow for a story.
//
// The workflowName must match a workflow defined in the configuration (e.g.,
// "analyze", "implement", "test"). The storyKey is substituted into the
// workflow's prompt template.
//
// Returns the exit code from Claude CLI (0 for success, non-zero for failure).
func (r *Runner) RunSingle(ctx context.Context, workflowName, storyKey string) int {
	prompt, err := r.config.GetPrompt(workflowName, storyKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	label := fmt.Sprintf("%s: %s", workflowName, storyKey)
	return r.runClaude(ctx, prompt, label)
}

// RunRaw executes an arbitrary prompt without template expansion.
//
// Use this method for one-off or custom prompts that don't correspond to
// configured workflows. The prompt is passed directly to Claude CLI.
//
// Returns the exit code from Claude CLI (0 for success, non-zero for failure).
func (r *Runner) RunRaw(ctx context.Context, prompt string) int {
	return r.runClaude(ctx, prompt, "raw")
}

// RunFullCycle executes all configured steps in sequence for a story.
//
// Deprecated: Use the lifecycle package for multi-step workflows with
// status-based routing instead.
//
// This method runs the complete development cycle (analyze, implement, test,
// etc.) as configured in full_cycle.steps. Each step is executed in order,
// and execution stops on the first failure.
//
// Output includes a cycle header, per-step progress, and a summary with
// timing information for all completed steps.
//
// Returns 0 if all steps succeed, or the exit code from the first failed step.
func (r *Runner) RunFullCycle(ctx context.Context, storyKey string) int {
	totalStart := time.Now()

	// Build steps from config
	stepNames := r.config.GetFullCycleSteps()
	steps := make([]Step, 0, len(stepNames))

	for _, name := range stepNames {
		prompt, err := r.config.GetPrompt(name, storyKey)
		if err != nil {
			fmt.Printf("Error building step %s: %v\n", name, err)
			return 1
		}
		steps = append(steps, Step{Name: name, Prompt: prompt})
	}

	r.printer.CycleHeader(storyKey)

	results := make([]output.StepResult, len(steps))

	for i, step := range steps {
		r.printer.StepStart(i+1, len(steps), step.Name)

		stepStart := time.Now()
		exitCode := r.runClaude(ctx, step.Prompt, fmt.Sprintf("%s: %s", step.Name, storyKey))
		duration := time.Since(stepStart)

		results[i] = output.StepResult{
			Name:     step.Name,
			Duration: duration,
			Success:  exitCode == 0,
		}

		if exitCode != 0 {
			r.printer.CycleFailed(storyKey, step.Name, time.Since(totalStart))
			return exitCode
		}

		fmt.Println() // Add spacing between steps
	}

	r.printer.CycleSummary(storyKey, results, time.Since(totalStart))
	return 0
}

// runClaude executes Claude CLI with the given prompt and handles streaming output.
//
// This is the core execution method used by all public Runner methods.
// It displays a command header, streams events to the printer via handleEvent,
// and displays a footer with timing and exit status.
func (r *Runner) runClaude(ctx context.Context, prompt, label string) int {
	r.printer.CommandHeader(label, prompt, r.config.Output.TruncateLength)

	startTime := time.Now()

	handler := func(event claude.Event) {
		r.handleEvent(event)
	}

	exitCode, err := r.executor.ExecuteWithResult(ctx, prompt, handler)
	if err != nil {
		fmt.Printf("Error executing claude: %v\n", err)
		exitCode = 1
	}

	duration := time.Since(startTime)
	r.printer.CommandFooter(duration, exitCode == 0, exitCode)

	return exitCode
}

// handleEvent routes a Claude streaming event to the appropriate printer method.
//
// Events are dispatched based on their type: session start/end, text output,
// tool usage, and tool results. Each event type is formatted differently
// by the printer for terminal display.
func (r *Runner) handleEvent(event claude.Event) {
	switch {
	case event.SessionStarted:
		r.printer.SessionStart()

	case event.IsText():
		r.printer.Text(event.Text)

	case event.IsToolUse():
		r.printer.ToolUse(event.ToolName, event.ToolDescription, event.ToolCommand, event.ToolFilePath)

	case event.IsToolResult():
		r.printer.ToolResult(event.ToolStdout, event.ToolStderr, r.config.Output.TruncateLines)

	case event.SessionComplete:
		r.printer.SessionEnd(0, true) // Duration handled elsewhere
	}
}
