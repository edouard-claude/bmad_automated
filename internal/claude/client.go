package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Executor runs Claude CLI and returns streaming events.
//
// Executor provides two execution modes:
//   - [Executor.Execute]: Fire-and-forget mode that returns a channel of [Event] objects.
//     Use this when you want to process events as they arrive but don't need the exit code.
//   - [Executor.ExecuteWithResult]: Blocking mode that processes events via an [EventHandler]
//     callback and returns the exit code. Use this for production workflows where you need
//     to know if Claude completed successfully.
//
// For testing, use [MockExecutor] which implements this interface without spawning processes.
type Executor interface {
	// Execute runs Claude with the given prompt and returns a channel of [Event] objects.
	// The channel is closed when Claude exits or the context is canceled.
	// Returns an error if Claude fails to start (e.g., binary not found).
	//
	// Note: This method is fire-and-forget; the exit status is not available.
	// Use [Executor.ExecuteWithResult] if you need the exit code.
	Execute(ctx context.Context, prompt string) (<-chan Event, error)

	// ExecuteWithResult runs Claude with the given prompt and waits for completion.
	// The handler is called for each [Event] received during execution.
	// Returns the exit code (0 for success) and any error encountered during execution.
	//
	// This is the recommended method for production use as it provides the exit code
	// needed to determine if Claude completed successfully.
	ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error)
}

// EventHandler is a callback function invoked for each [Event] received from Claude.
//
// The handler is called synchronously in the order events are received. Handlers
// should process events quickly to avoid blocking the event stream. For time-consuming
// processing, consider queuing events for async handling.
//
// The handler receives events of all types (system, assistant, user, result).
// Use the [Event] convenience methods ([Event.IsText], [Event.IsToolUse], [Event.IsToolResult])
// to filter for specific event types.
type EventHandler func(event Event)

// ExecutorConfig contains configuration for creating a [DefaultExecutor].
//
// All fields are optional and have sensible defaults. Use [NewExecutor] to create
// an executor with this configuration.
type ExecutorConfig struct {
	// BinaryPath is the path to the Claude CLI binary.
	// If empty, defaults to "claude" which must be in PATH.
	// Set this to an absolute path if Claude is installed in a non-standard location.
	BinaryPath string

	// OutputFormat is the Claude CLI output format flag.
	// If empty, defaults to "stream-json" which is required for event parsing.
	// This should not normally be changed.
	OutputFormat string

	// Parser is the JSON parser used to parse Claude's streaming output.
	// If nil, a [DefaultParser] is created with default settings.
	// Provide a custom parser only if you need to adjust buffer sizes.
	Parser Parser

	// StderrHandler is called for each line written to stderr by Claude.
	// If nil, stderr output is silently discarded.
	// Set this to capture error messages or debug output from Claude.
	StderrHandler func(line string)
}

// DefaultExecutor implements [Executor] by spawning Claude as a subprocess.
//
// This is the production implementation that uses os/exec to run the Claude CLI.
// It captures stdout for event parsing and optionally handles stderr via the
// configured [ExecutorConfig.StderrHandler].
//
// Create instances using [NewExecutor] rather than constructing directly.
type DefaultExecutor struct {
	config ExecutorConfig
	parser Parser
}

// NewExecutor creates a new [DefaultExecutor] with the given configuration.
//
// Default values are applied for any unset configuration fields:
//   - BinaryPath defaults to "claude"
//   - OutputFormat defaults to "stream-json"
//   - Parser defaults to a new [DefaultParser]
//
// Pass an empty [ExecutorConfig] to use all defaults.
func NewExecutor(config ExecutorConfig) *DefaultExecutor {
	if config.BinaryPath == "" {
		config.BinaryPath = "claude"
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "stream-json"
	}

	parser := config.Parser
	if parser == nil {
		parser = NewParser()
	}

	return &DefaultExecutor{
		config: config,
		parser: parser,
	}
}

// Execute runs Claude with the given prompt and returns a channel of [Event] objects.
//
// The returned channel emits events as they are parsed from Claude's streaming output.
// The channel is closed when:
//   - Claude exits normally
//   - The context is canceled
//   - An unrecoverable error occurs
//
// Note: This method does not provide the exit status. The command's exit code is
// intentionally not propagated. Use [DefaultExecutor.ExecuteWithResult] if you need
// to check whether Claude completed successfully.
func (e *DefaultExecutor) Execute(ctx context.Context, prompt string) (<-chan Event, error) {
	cmd := exec.CommandContext(ctx, e.config.BinaryPath,
		"--dangerously-skip-permissions",
		"--output-format", e.config.OutputFormat,
		"--verbose",
		"-p",
		prompt,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Handle stderr in background (no synchronization for fire-and-forget mode)
	go e.handleStderr(stderr, nil)

	// Parse stdout and return events channel
	events := e.parser.Parse(stdout)

	// Wait for command completion in background.
	// Note: Exit status is intentionally not propagated; use ExecuteWithResult if needed.
	go func() {
		_ = cmd.Wait() //nolint:errcheck // Exit status intentionally ignored; use ExecuteWithResult if needed
	}()

	return events, nil
}

// ExecuteWithResult runs Claude with the given prompt and waits for completion.
//
// This is the recommended method for production use. It processes events via the
// provided [EventHandler] callback and returns the exit code when Claude completes.
//
// Exit code semantics:
//   - 0: Claude completed successfully
//   - Non-zero: Claude exited with an error (check stderr via [ExecutorConfig.StderrHandler])
//
// The handler may be nil if you only need the exit code without processing events.
// If the handler is provided, it is called synchronously for each event before
// this method returns.
func (e *DefaultExecutor) ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error) {
	cmd := exec.CommandContext(ctx, e.config.BinaryPath,
		"--dangerously-skip-permissions",
		"--output-format", e.config.OutputFormat,
		"--verbose",
		"-p",
		prompt,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start claude: %w", err)
	}

	// Handle stderr in background with synchronization
	var stderrWg sync.WaitGroup
	stderrWg.Add(1)
	go e.handleStderr(stderr, &stderrWg)

	// Process events with context cancellation check
	events := e.parser.Parse(stdout)
eventLoop:
	for {
		select {
		case <-ctx.Done():
			break eventLoop
		case event, ok := <-events:
			if !ok {
				break eventLoop
			}
			if handler != nil {
				handler(event)
			}
		}
	}

	// Wait for stderr to be fully read
	stderrWg.Wait()

	// Wait for command completion
	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, err
		}
	}

	return exitCode, nil
}

func (e *DefaultExecutor) handleStderr(stderr io.ReadCloser, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	if e.config.StderrHandler == nil {
		_, _ = io.Copy(io.Discard, stderr) //nolint:errcheck // Intentionally discarding stderr
		return
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		e.config.StderrHandler(scanner.Text())
	}
}

// MockExecutor implements [Executor] for testing without spawning real processes.
//
// Configure the mock by setting its fields before calling Execute or ExecuteWithResult:
//
//	mock := &MockExecutor{
//	    Events: []Event{{Type: EventTypeAssistant, Text: "Hello"}},
//	    ExitCode: 0,
//	}
//	exitCode, err := mock.ExecuteWithResult(ctx, "prompt", handler)
//
// After execution, check RecordedPrompts to verify the prompts that were passed:
//
//	if len(mock.RecordedPrompts) != 1 || mock.RecordedPrompts[0] != "expected prompt" {
//	    t.Error("unexpected prompt")
//	}
//
// To simulate errors, set the Error field:
//
//	mock := &MockExecutor{Error: errors.New("connection failed")}
type MockExecutor struct {
	// Events is the list of [Event] objects to emit during execution.
	// These are sent in order to the events channel or handler.
	Events []Event

	// Error is returned from Execute/ExecuteWithResult if non-nil.
	// When set, no events are emitted.
	Error error

	// ExitCode is the value returned from [MockExecutor.ExecuteWithResult].
	// Ignored if Error is set.
	ExitCode int

	// RecordedPrompts accumulates all prompts passed to Execute/ExecuteWithResult.
	// Use this in tests to verify the correct prompts were sent.
	RecordedPrompts []string
}

// Execute returns the pre-configured [MockExecutor.Events] via a channel.
//
// The prompt is recorded in [MockExecutor.RecordedPrompts] for later verification.
// If [MockExecutor.Error] is set, it returns nil and the error immediately.
// Otherwise, events are sent to the channel asynchronously and the channel is
// closed when all events have been sent or the context is canceled.
func (m *MockExecutor) Execute(ctx context.Context, prompt string) (<-chan Event, error) {
	m.RecordedPrompts = append(m.RecordedPrompts, prompt)

	if m.Error != nil {
		return nil, m.Error
	}

	events := make(chan Event)
	go func() {
		defer close(events)
		for _, event := range m.Events {
			select {
			case <-ctx.Done():
				return
			case events <- event:
			}
		}
	}()

	return events, nil
}

// ExecuteWithResult returns the pre-configured [MockExecutor.ExitCode].
//
// The prompt is recorded in [MockExecutor.RecordedPrompts] for later verification.
// If [MockExecutor.Error] is set, it returns 1 and the error immediately.
// Otherwise, all [MockExecutor.Events] are passed to the handler synchronously,
// then the configured exit code is returned.
func (m *MockExecutor) ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error) {
	m.RecordedPrompts = append(m.RecordedPrompts, prompt)

	if m.Error != nil {
		return 1, m.Error
	}

	for _, event := range m.Events {
		if handler != nil {
			handler(event)
		}
	}

	return m.ExitCode, nil
}
