package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edouard-claude/bmad_automated/internal/output"
	"github.com/edouard-claude/bmad_automated/internal/router"
	"github.com/edouard-claude/bmad_automated/internal/status"
)

// StatusReader provides story status lookup for workflow routing.
//
// This interface is intentionally duplicated from the lifecycle package
// to maintain dependency inversion - the workflow package should not import
// lifecycle. Implementations typically read status from YAML story files.
type StatusReader interface {
	// GetStoryStatus returns the current status of a story.
	GetStoryStatus(storyKey string) (status.Status, error)
}

// QueueRunner processes multiple stories in sequence with status-based routing.
//
// QueueRunner wraps a [Runner] to enable batch processing of stories. For each
// story, it looks up the current status via [StatusReader], routes to the
// appropriate workflow using the router package, and executes the workflow.
//
// Use [NewQueueRunner] to create a QueueRunner instance.
type QueueRunner struct {
	runner *Runner
}

// NewQueueRunner creates a new queue runner wrapping the given [Runner].
//
// The provided runner is used to execute individual workflows for each story.
func NewQueueRunner(runner *Runner) *QueueRunner {
	return &QueueRunner{runner: runner}
}

// RunQueueWithStatus executes the appropriate workflow for each story based on its status.
//
// For each story in storyKeys, the method:
//  1. Looks up the story's current status via statusReader
//  2. Routes to the appropriate workflow based on status (via router.GetWorkflow)
//  3. Executes the workflow using [Runner.RunSingle]
//
// Behavior:
//   - Stories with "done" status are skipped (counted as successful)
//   - Processing stops on the first workflow failure (fail-fast)
//   - Unknown status values cause immediate failure
//
// Output includes a queue header listing all stories, per-story progress
// indicators, and a summary with timing and success/failure counts.
//
// Returns 0 if all stories complete successfully or are skipped, or the exit
// code from the first failed story.
func (q *QueueRunner) RunQueueWithStatus(ctx context.Context, storyKeys []string, statusReader StatusReader) int {
	queueStart := time.Now()
	results := make([]output.StoryResult, 0, len(storyKeys))

	q.runner.printer.QueueHeader(len(storyKeys), storyKeys)

	for i, storyKey := range storyKeys {
		q.runner.printer.QueueStoryStart(i+1, len(storyKeys), storyKey)

		storyStart := time.Now()

		// Get story status
		storyStatus, err := statusReader.GetStoryStatus(storyKey)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			result := output.StoryResult{
				Key:      storyKey,
				Success:  false,
				Duration: time.Since(storyStart),
				FailedAt: "status",
			}
			results = append(results, result)
			q.runner.printer.QueueSummary(results, storyKeys, time.Since(queueStart))
			return 1
		}

		// Route to appropriate workflow
		workflowName, err := router.GetWorkflow(storyStatus)
		if err != nil {
			if errors.Is(err, router.ErrStoryComplete) {
				// Done stories are skipped, not failures
				fmt.Printf("  ↷ Skipped (already done)\n")
				result := output.StoryResult{
					Key:      storyKey,
					Success:  true,
					Duration: time.Since(storyStart),
					Skipped:  true,
				}
				results = append(results, result)
				fmt.Println() // Add spacing between stories
				continue
			}
			if errors.Is(err, router.ErrUnknownStatus) {
				fmt.Printf("  Error: unknown status value: %s\n", storyStatus)
			} else {
				fmt.Printf("  Error: %v\n", err)
			}
			result := output.StoryResult{
				Key:      storyKey,
				Success:  false,
				Duration: time.Since(storyStart),
				FailedAt: "routing",
			}
			results = append(results, result)
			q.runner.printer.QueueSummary(results, storyKeys, time.Since(queueStart))
			return 1
		}

		// Run the workflow
		exitCode := q.runner.RunSingle(ctx, workflowName, storyKey)
		duration := time.Since(storyStart)

		result := output.StoryResult{
			Key:      storyKey,
			Success:  exitCode == 0,
			Duration: duration,
		}

		if exitCode != 0 {
			result.FailedAt = workflowName
			results = append(results, result)
			q.runner.printer.QueueSummary(results, storyKeys, time.Since(queueStart))
			return exitCode
		}

		results = append(results, result)
		fmt.Println() // Add spacing between stories
	}

	q.runner.printer.QueueSummary(results, storyKeys, time.Since(queueStart))
	return 0
}
