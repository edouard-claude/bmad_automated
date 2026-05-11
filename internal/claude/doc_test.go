package claude_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/edouard-claude/bmad_automated/internal/claude"
)

// Example_mockExecutor demonstrates using MockExecutor for testing Claude
// integrations without spawning real processes.
func Example_mockExecutor() {
	// Configure mock with predefined events
	mock := &claude.MockExecutor{
		Events: []claude.Event{
			{Type: claude.EventTypeSystem, SessionStarted: true},
			{Type: claude.EventTypeAssistant, Text: "I'll help you with that task."},
			{Type: claude.EventTypeResult, SessionComplete: true},
		},
		ExitCode: 0,
	}

	// Execute with handler
	exitCode, err := mock.ExecuteWithResult(
		context.Background(),
		"Analyze this code",
		func(event claude.Event) {
			if event.IsText() {
				fmt.Println(event.Text)
			}
		},
	)

	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("exit code:", exitCode)
	// Output:
	// I'll help you with that task.
	// exit code: 0
}

// Example_parseSingle demonstrates parsing a single JSON line from Claude's
// streaming output into an Event.
func Example_parseSingle() {
	// Parse a text event from Claude's stream-json format
	jsonLine := `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello, world!"}]}}`

	event, err := claude.ParseSingle(jsonLine)
	if err != nil {
		fmt.Println("parse error:", err)
		return
	}

	fmt.Println("type:", event.Type)
	fmt.Println("text:", event.Text)
	fmt.Println("is text:", event.IsText())
	// Output:
	// type: assistant
	// text: Hello, world!
	// is text: true
}

// Example_eventTypeChecking demonstrates using Event convenience methods to
// identify different event types from Claude's streaming output.
func Example_eventTypeChecking() {
	// Text output from Claude
	textEvent := claude.Event{
		Type: claude.EventTypeAssistant,
		Text: "Working on your request...",
	}
	fmt.Println("text event IsText:", textEvent.IsText())
	fmt.Println("text event IsToolUse:", textEvent.IsToolUse())

	// Tool invocation event
	toolEvent := claude.Event{
		Type:            claude.EventTypeAssistant,
		ToolName:        "Bash",
		ToolCommand:     "ls -la",
		ToolDescription: "List files",
	}
	fmt.Println("tool event IsText:", toolEvent.IsText())
	fmt.Println("tool event IsToolUse:", toolEvent.IsToolUse())

	// Tool result event
	resultEvent := claude.Event{
		Type:       claude.EventTypeUser,
		ToolStdout: "file1.go\nfile2.go",
	}
	fmt.Println("result event IsToolResult:", resultEvent.IsToolResult())
	// Output:
	// text event IsText: true
	// text event IsToolUse: false
	// tool event IsText: false
	// tool event IsToolUse: true
	// result event IsToolResult: true
}

// Example_parser demonstrates using the Parser interface to process
// streaming JSON output from Claude CLI.
func Example_parser() {
	// Simulate Claude's streaming JSON output
	jsonOutput := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[{"type":"text","text":"Starting analysis..."}]}}
{"type":"result"}`

	parser := claude.NewParser()
	events := parser.Parse(strings.NewReader(jsonOutput))

	// Process events as they arrive
	for event := range events {
		switch {
		case event.SessionStarted:
			fmt.Println("session started")
		case event.IsText():
			fmt.Println("text:", event.Text)
		case event.SessionComplete:
			fmt.Println("session complete")
		}
	}
	// Output:
	// session started
	// text: Starting analysis...
	// session complete
}
