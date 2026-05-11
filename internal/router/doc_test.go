package router_test

import (
	"errors"
	"fmt"

	"github.com/edouard-claude/bmad_automated/internal/router"
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// This example demonstrates using GetWorkflow to map a story status
// to its corresponding workflow name. This is used by single-step
// commands like "run --workflow auto".
func Example_getWorkflow() {
	// GetWorkflow returns the single workflow for a given status.
	// This is the mapping used by the run command for single-step execution.

	// Backlog stories need to be created first
	workflow, err := router.GetWorkflow(status.StatusBacklog)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Backlog workflow:", workflow)

	// Ready-for-dev and in-progress stories need development
	workflow, _ = router.GetWorkflow(status.StatusReadyForDev)
	fmt.Println("Ready-for-dev workflow:", workflow)

	// Review stories need code review
	workflow, _ = router.GetWorkflow(status.StatusReview)
	fmt.Println("Review workflow:", workflow)

	// Done stories return ErrStoryComplete (not an error, just complete)
	_, err = router.GetWorkflow(status.StatusDone)
	fmt.Println("Done story complete:", errors.Is(err, router.ErrStoryComplete))
	// Output:
	// Backlog workflow: create-story
	// Ready-for-dev workflow: dev-story
	// Review workflow: code-review
	// Done story complete: true
}

// This example demonstrates using GetLifecycle to retrieve the full
// sequence of steps needed to complete a story from its current status.
// This is used by the lifecycle executor for multi-step execution.
func Example_getLifecycle() {
	// GetLifecycle returns all remaining steps to complete a story.
	// Each step includes the workflow to run and the next status to set.

	// A backlog story needs all steps: create -> dev -> review -> commit
	steps, err := router.GetLifecycle(status.StatusBacklog)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Backlog steps:", len(steps))
	fmt.Println("First step workflow:", steps[0].Workflow)

	// A story in review only needs review and commit steps
	steps, _ = router.GetLifecycle(status.StatusReview)
	fmt.Println("Review steps:", len(steps))
	fmt.Println("First step workflow:", steps[0].Workflow)

	// Done stories return ErrStoryComplete
	_, err = router.GetLifecycle(status.StatusDone)
	fmt.Println("Done story complete:", errors.Is(err, router.ErrStoryComplete))
	// Output:
	// Backlog steps: 4
	// First step workflow: create-story
	// Review steps: 2
	// First step workflow: code-review
	// Done story complete: true
}
