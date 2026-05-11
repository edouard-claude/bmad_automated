package lifecycle_test

import (
	"context"
	"fmt"

	"github.com/edouard-claude/bmad_automated/internal/lifecycle"
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// mockWorkflowRunner implements lifecycle.WorkflowRunner for examples.
type mockWorkflowRunner struct {
	exitCode int
}

func (m *mockWorkflowRunner) RunSingle(ctx context.Context, workflowName, storyKey string) int {
	fmt.Printf("Running workflow: %s for %s\n", workflowName, storyKey)
	return m.exitCode
}

// mockStatusReader implements lifecycle.StatusReader for examples.
type mockStatusReader struct {
	status status.Status
}

func (m *mockStatusReader) GetStoryStatus(storyKey string) (status.Status, error) {
	return m.status, nil
}

// mockStatusWriter implements lifecycle.StatusWriter for examples.
type mockStatusWriter struct{}

func (m *mockStatusWriter) UpdateStatus(storyKey string, newStatus status.Status) error {
	fmt.Printf("Status updated: %s -> %s\n", storyKey, newStatus)
	return nil
}

// Example_executor demonstrates using the lifecycle Executor to run a story
// through its complete workflow sequence based on current status.
func Example_executor() {
	// Create mock dependencies
	runner := &mockWorkflowRunner{exitCode: 0}
	reader := &mockStatusReader{status: status.StatusReview}
	writer := &mockStatusWriter{}

	// Create executor with dependencies
	executor := lifecycle.NewExecutor(runner, reader, writer)

	// Execute lifecycle from current status (review -> done)
	err := executor.Execute(context.Background(), "EPIC-1-story")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Lifecycle complete")
	// Output:
	// Running workflow: code-review for EPIC-1-story
	// Status updated: EPIC-1-story -> done
	// Running workflow: git-commit for EPIC-1-story
	// Status updated: EPIC-1-story -> done
	// Lifecycle complete
}

// Example_progressCallback demonstrates using SetProgressCallback to track
// workflow execution progress during lifecycle runs.
func Example_progressCallback() {
	// Create mock dependencies
	runner := &mockWorkflowRunner{exitCode: 0}
	reader := &mockStatusReader{status: status.StatusReview}
	writer := &mockStatusWriter{}

	// Create executor
	executor := lifecycle.NewExecutor(runner, reader, writer)

	// Set progress callback for UI updates
	executor.SetProgressCallback(func(stepIndex, totalSteps int, workflow string) {
		fmt.Printf("Progress: step %d/%d - %s\n", stepIndex, totalSteps, workflow)
	})

	// Execute lifecycle
	_ = executor.Execute(context.Background(), "EPIC-1-story")
	// Output:
	// Progress: step 1/2 - code-review
	// Running workflow: code-review for EPIC-1-story
	// Status updated: EPIC-1-story -> done
	// Progress: step 2/2 - git-commit
	// Running workflow: git-commit for EPIC-1-story
	// Status updated: EPIC-1-story -> done
}

// Example_getSteps demonstrates using GetSteps to preview the remaining
// workflow steps without executing them (dry-run functionality).
func Example_getSteps() {
	// Create mock dependencies
	runner := &mockWorkflowRunner{exitCode: 0}
	reader := &mockStatusReader{status: status.StatusReadyForDev}
	writer := &mockStatusWriter{}

	// Create executor
	executor := lifecycle.NewExecutor(runner, reader, writer)

	// Get steps without executing (dry-run)
	steps, err := executor.GetSteps("EPIC-1-story")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("Remaining steps:")
	for i, step := range steps {
		fmt.Printf("  %d. %s -> %s\n", i+1, step.Workflow, step.NextStatus)
	}
	// Output:
	// Remaining steps:
	//   1. dev-story -> review
	//   2. code-review -> done
	//   3. git-commit -> done
}
