package router

import (
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// LifecycleStep represents a single step in the story lifecycle sequence.
//
// Each step contains the workflow to execute and the status to transition to
// after the workflow completes successfully. The lifecycle executor uses these
// steps to drive a story from its current status through to completion.
type LifecycleStep struct {
	// Workflow is the name of the workflow to execute for this step.
	// Must correspond to a key in the workflows configuration.
	Workflow string

	// NextStatus is the status to set after this step completes successfully.
	// The final step typically sets status to "done".
	NextStatus status.Status
}

// GetLifecycle returns the complete sequence of lifecycle steps from the given
// status through to "done".
//
// This is the multi-step router used by the lifecycle executor to run a story
// through its full lifecycle. Unlike [GetWorkflow] which returns a single workflow,
// GetLifecycle returns all remaining steps needed to complete the story.
//
// The sequences are:
//   - backlog: create-story -> dev-story -> code-review -> git-commit -> done
//   - ready-for-dev, in-progress: dev-story -> code-review -> git-commit -> done
//   - review: code-review -> git-commit -> done
//   - done: [ErrStoryComplete]
//
// Returns [ErrStoryComplete] for done stories (caller should skip, not fail).
// Returns [ErrUnknownStatus] for unrecognized status values (likely YAML typo).
//
// See [status.Status] for valid status values.
func GetLifecycle(s status.Status) ([]LifecycleStep, error) {
	switch s {
	case status.StatusBacklog:
		return []LifecycleStep{
			{Workflow: "create-story", NextStatus: status.StatusReadyForDev},
			{Workflow: "dev-story", NextStatus: status.StatusReview},
			{Workflow: "code-review", NextStatus: status.StatusDone},
			{Workflow: "git-commit", NextStatus: status.StatusDone},
		}, nil
	case status.StatusReadyForDev, status.StatusInProgress:
		return []LifecycleStep{
			{Workflow: "dev-story", NextStatus: status.StatusReview},
			{Workflow: "code-review", NextStatus: status.StatusDone},
			{Workflow: "git-commit", NextStatus: status.StatusDone},
		}, nil
	case status.StatusReview:
		return []LifecycleStep{
			{Workflow: "code-review", NextStatus: status.StatusDone},
			{Workflow: "git-commit", NextStatus: status.StatusDone},
		}, nil
	case status.StatusDone:
		return nil, ErrStoryComplete
	default:
		return nil, ErrUnknownStatus
	}
}
