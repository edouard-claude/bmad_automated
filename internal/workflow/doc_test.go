package workflow_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/edouard-claude/bmad_automated/internal/claude"
	"github.com/edouard-claude/bmad_automated/internal/config"
	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/workflow"
)

// Example_runner demonstrates using Runner to execute a single workflow
// with Claude CLI integration and formatted output.
func Example_runner() {
	// Create a mock executor for testing (use claude.NewExecutor in production)
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "Analyzing the story requirements..."},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}

	// Capture output for demonstration
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	cfg := config.DefaultConfig()

	// Create and run the workflow
	runner := workflow.NewRunner(mockExecutor, printer, cfg)
	exitCode := runner.RunSingle(context.Background(), "create-story", "EPIC-1-story")

	// Check the recorded prompt was correct
	if len(mockExecutor.RecordedPrompts) > 0 {
		fmt.Println("prompt contains story key:", contains(mockExecutor.RecordedPrompts[0], "EPIC-1-story"))
	}
	fmt.Println("exit code:", exitCode)
	// Output:
	// prompt contains story key: true
	// exit code: 0
}

// Example_eventHandler demonstrates how Runner processes Claude events
// and routes them to the appropriate printer methods.
func Example_eventHandler() {
	// Configure mock with different event types
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "Let me analyze this code."},
			{Type: claude.EventTypeAssistant, ToolName: "Read", ToolFilePath: "/path/to/file.go"},
			{Type: claude.EventTypeUser, ToolStdout: "package main\n\nfunc main() {}"},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}

	// Capture output
	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	cfg := config.DefaultConfig()

	// Execute workflow
	runner := workflow.NewRunner(mockExecutor, printer, cfg)
	_ = runner.RunRaw(context.Background(), "analyze this code")

	// Verify output contains expected elements
	out := buf.String()
	fmt.Println("has session start:", contains(out, "Session started"))
	fmt.Println("has text output:", contains(out, "analyze this code"))
	fmt.Println("has tool use:", contains(out, "Read"))
	// Output:
	// has session start: true
	// has text output: true
	// has tool use: true
}

// Example_runRaw demonstrates executing an arbitrary prompt without
// template expansion, useful for custom one-off commands.
func Example_runRaw() {
	mockExecutor := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "Done!"},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}

	buf := &bytes.Buffer{}
	printer := output.NewPrinterWithWriter(buf)
	cfg := config.DefaultConfig()

	runner := workflow.NewRunner(mockExecutor, printer, cfg)
	exitCode := runner.RunRaw(context.Background(), "List all Go files in this project")

	fmt.Println("recorded prompt:", mockExecutor.RecordedPrompts[0])
	fmt.Println("exit code:", exitCode)
	// Output:
	// recorded prompt: List all Go files in this project
	// exit code: 0
}

// contains is a helper for checking string containment.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
