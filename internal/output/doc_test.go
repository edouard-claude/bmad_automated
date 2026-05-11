package output_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/edouard-claude/bmad_automated/internal/output"
)

// Example_printer demonstrates the Printer interface methods for terminal output.
//
// The Printer interface provides structured output operations for session lifecycle,
// step progress, tool usage display, and summaries.
func Example_printer() {
	// Create a printer with buffer for capture (production uses NewPrinter())
	var buf bytes.Buffer
	printer := output.NewPrinterWithWriter(&buf)

	// Session lifecycle methods
	printer.SessionStart()
	// captures: "● Session started"

	// Step progress methods
	printer.StepStart(1, 4, "create-story")
	// captures: "[1/4] create-story"

	// Verify printer captured output
	if buf.Len() > 0 {
		fmt.Println("printer output captured")
	}
	// Output:
	// printer output captured
}

// Example_styles demonstrates using the styled output methods.
//
// The DefaultPrinter uses lipgloss styles for consistent terminal formatting
// including headers, success/error indicators, and dividers.
func Example_styles() {
	var buf bytes.Buffer
	printer := output.NewPrinterWithWriter(&buf)

	// Divider creates a visual separator
	printer.Divider()

	// Text displays plain text from Claude
	printer.Text("Processing your request...")

	// Tool usage shows Claude's tool invocations
	printer.ToolUse("Bash", "List files", "ls -la", "")

	// Check output was captured
	if buf.Len() > 0 {
		fmt.Println("output captured:", buf.Len() > 0)
	}
	// Output:
	// output captured: true
}

// Example_testCapture demonstrates capturing output in tests using NewPrinterWithWriter.
//
// This pattern is essential for testing code that uses the Printer interface
// without writing to actual stdout.
func Example_testCapture() {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create printer with custom writer
	printer := output.NewPrinterWithWriter(&buf)

	// All output goes to the buffer instead of stdout
	printer.Text("Hello from test")
	printer.Divider()

	// Verify output was captured
	output := buf.String()
	if len(output) > 0 {
		fmt.Println("captured bytes:", len(output) > 0)
		fmt.Println("contains 'Hello':", contains(output, "Hello"))
	}
	// Output:
	// captured bytes: true
	// contains 'Hello': true
}

// contains is a helper for the example.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))
}

// Example_commandOutput demonstrates the command header/footer pattern.
//
// Commands show a header before execution with the prompt, and a footer
// after completion with duration and success/failure status.
func Example_commandOutput() {
	var buf bytes.Buffer
	printer := output.NewPrinterWithWriter(&buf)

	// Header shows command label and prompt
	printer.CommandHeader("create-story", "Create story 7-1-define-schema", 80)

	// ... command execution happens here ...

	// Footer shows result
	duration := 5 * time.Second
	printer.CommandFooter(duration, true, 0)

	if buf.Len() > 0 {
		fmt.Println("command output captured")
	}
	// Output:
	// command output captured
}

// Example_stepResult demonstrates the StepResult type for cycle summaries.
func Example_stepResult() {
	// StepResult tracks individual workflow step execution
	step := output.StepResult{
		Name:     "create-story",
		Duration: 2 * time.Second,
		Success:  true,
	}

	fmt.Println("step:", step.Name)
	fmt.Println("success:", step.Success)
	fmt.Println("duration:", step.Duration)
	// Output:
	// step: create-story
	// success: true
	// duration: 2s
}

// Example_storyResult demonstrates the StoryResult type for queue summaries.
func Example_storyResult() {
	// StoryResult tracks individual story processing in queue/epic operations
	result := output.StoryResult{
		Key:      "7-1-define-schema",
		Success:  true,
		Duration: 30 * time.Second,
		FailedAt: "",
		Skipped:  false,
	}

	fmt.Println("key:", result.Key)
	fmt.Println("success:", result.Success)
	fmt.Println("skipped:", result.Skipped)
	// Output:
	// key: 7-1-define-schema
	// success: true
	// skipped: false
}

// Example_cycleSummary demonstrates the cycle summary output.
func Example_cycleSummary() {
	var buf bytes.Buffer
	printer := output.NewPrinterWithWriter(&buf)

	steps := []output.StepResult{
		{Name: "create-story", Duration: 2 * time.Second, Success: true},
		{Name: "dev-story", Duration: 10 * time.Second, Success: true},
		{Name: "code-review", Duration: 5 * time.Second, Success: true},
	}

	printer.CycleSummary("7-1-story", steps, 17*time.Second)

	if buf.Len() > 0 {
		fmt.Println("cycle summary captured")
	}
	// Output:
	// cycle summary captured
}
