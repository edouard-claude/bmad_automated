package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edouard-claude/bmad_automated/internal/lifecycle"
	"github.com/edouard-claude/bmad_automated/internal/router"
)

func newEpicCommand(app *App) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "epic <epic-id>",
		Short: "Run full lifecycle for all stories in an epic",
		Long: `Run the complete lifecycle for all stories in an epic to completion.

Finds all stories matching the pattern {epic-id}-{N}-* where N is numeric,
sorts them by story number, and runs each to completion before moving to the next.

For each story, executes all remaining workflows based on its current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → skipped (story already complete)

The epic command stops on the first failure. Done stories are skipped and do not cause failure.
Status is updated in sprint-status.yaml after each successful workflow.

Use --dry-run to preview workflows without executing them.

Example:
  bmad-automate epic 6
  # Runs 6-1-*, 6-2-*, 6-3-*, etc. each to completion in order`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			epicID := args[0]

			// Get all stories for this epic
			storyKeys, err := app.StatusReader.GetEpicStories(epicID)
			if err != nil {
				cmd.SilenceUsage = true
				return NewExitError(1)
			}

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				return runEpicDryRun(cmd, executor, epicID, storyKeys)
			}

			// Execute full lifecycle for each story in order
			for _, storyKey := range storyKeys {
				err := executor.Execute(ctx, storyKey)
				if err != nil {
					cmd.SilenceUsage = true
					if errors.Is(err, router.ErrStoryComplete) {
						fmt.Printf("Story %s is already complete, skipping\n", storyKey)
						continue
					}
					fmt.Printf("Error running lifecycle for story %s: %v\n", storyKey, err)
					return NewExitError(1)
				}
				fmt.Printf("Story %s completed successfully\n", storyKey)
			}

			fmt.Printf("All %d stories processed\n", len(storyKeys))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")

	return cmd
}

func runEpicDryRun(cmd *cobra.Command, executor *lifecycle.Executor, epicID string, storyKeys []string) error {
	fmt.Printf("Dry run for epic %s:\n", epicID)

	totalWorkflows := 0
	storiesWithWork := 0
	storiesComplete := 0

	for _, storyKey := range storyKeys {
		fmt.Println()
		fmt.Printf("Story %s:\n", storyKey)

		steps, err := executor.GetSteps(storyKey)
		if err != nil {
			if errors.Is(err, router.ErrStoryComplete) {
				fmt.Printf("  (already complete)\n")
				storiesComplete++
				continue
			}
			cmd.SilenceUsage = true
			fmt.Printf("  Error: %v\n", err)
			return NewExitError(1)
		}

		for i, step := range steps {
			fmt.Printf("  %d. %s → %s\n", i+1, step.Workflow, step.NextStatus)
		}
		totalWorkflows += len(steps)
		storiesWithWork++
	}

	fmt.Println()
	if storiesComplete > 0 {
		fmt.Printf("Total: %d workflows across %d stories (%d already complete)\n", totalWorkflows, storiesWithWork, storiesComplete)
	} else {
		fmt.Printf("Total: %d workflows across %d stories\n", totalWorkflows, storiesWithWork)
	}

	return nil
}
