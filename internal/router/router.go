// Package router provides workflow routing based on story status.
//
// The router maps story status values to workflow names for single-step execution
// and provides lifecycle step sequences for multi-step execution. It serves as
// the central decision point for determining which workflow to run for a given story.
//
// Key functions:
//   - [GetWorkflow] returns the single workflow for a status (used by run command)
//   - [GetLifecycle] returns the full step sequence to completion (used by lifecycle executor)
//
// Key types:
//   - [LifecycleStep] represents a single step in a lifecycle sequence
package router

import (
	"errors"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

// Sentinel errors for workflow routing.
var (
	// ErrStoryComplete is a sentinel error indicating the story has status "done"
	// and no workflow is needed. Callers should skip the story rather than treat
	// this as a failure condition.
	ErrStoryComplete = errors.New("story is complete, no workflow needed")

	// ErrUnknownStatus is a sentinel error indicating the status value is not
	// recognized. Callers should report this as an error, as it likely indicates
	// a typo in the sprint-status.yaml file.
	ErrUnknownStatus = errors.New("unknown status value")
)

// GetWorkflow returns the single workflow name for the given story status.
//
// This is the single-step router used by commands that execute one workflow at a time.
// The mapping is:
//   - backlog -> "create-story"
//   - ready-for-dev, in-progress -> "dev-story"
//   - review -> "code-review"
//   - done -> [ErrStoryComplete]
//
// Returns [ErrStoryComplete] for done stories (caller should skip, not fail).
// Returns [ErrUnknownStatus] for unrecognized status values (likely YAML typo).
//
// See [status.Status] for valid status values.
func GetWorkflow(s status.Status) (string, error) {
	switch s {
	case status.StatusBacklog:
		return "create-story", nil
	case status.StatusReadyForDev, status.StatusInProgress:
		return "dev-story", nil
	case status.StatusReview:
		return "code-review", nil
	case status.StatusDone:
		return "", ErrStoryComplete
	default:
		return "", ErrUnknownStatus
	}
}
