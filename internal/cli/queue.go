package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"bmad-automate/internal/lifecycle"
	"bmad-automate/internal/router"
	"bmad-automate/internal/status"
)

func newQueueCommand(app *App) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "queue <story-key|status> [story-key...]",
		Short: "Run full lifecycle for multiple stories",
		Long: `Run the complete lifecycle for multiple stories to completion.

Each story is run to completion before moving to the next.

You can pass explicit story keys, or a single status name to process all
stories with that status (backlog, ready-for-dev, in-progress, review):
  bmad-automate queue 6-5 6-6 6-7 6-8
  bmad-automate queue backlog

For each story, executes all remaining workflows based on its current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → skipped (story already complete)

The queue stops on the first failure. Done stories are skipped and do not cause failure.
Status is updated in sprint-status.yaml after each successful workflow.

Use --dry-run to preview workflows without executing them.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// If single arg is a valid status name, resolve to matching story keys
			storyKeys, err := resolveQueueArgs(args, app.StatusReader)
			if err != nil {
				cmd.SilenceUsage = true
				return err
			}
			args = storyKeys

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				return runQueueDryRun(cmd, executor, args)
			}

			// Execute full lifecycle for each story in order
			for _, storyKey := range args {
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

			fmt.Printf("All %d stories processed\n", len(args))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")

	return cmd
}

func runQueueDryRun(cmd *cobra.Command, executor *lifecycle.Executor, storyKeys []string) error {
	fmt.Printf("Dry run for %d stories:\n", len(storyKeys))

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

// resolveQueueArgs checks if a single argument is a valid status name.
// If so, it reads the sprint status and returns all story keys with that status,
// sorted by epic then story number. Otherwise, returns args unchanged.
func resolveQueueArgs(args []string, reader StatusReader) ([]string, error) {
	if len(args) != 1 {
		return args, nil
	}

	candidate := status.Status(args[0])
	if !candidate.IsValid() {
		return args, nil
	}

	sprintStatus, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	type storySort struct {
		key      string
		epicNum  int
		storyNum int
	}

	var matches []storySort
	for key, st := range sprintStatus.DevelopmentStatus {
		if st != candidate {
			continue
		}

		// Only include actual stories (pattern: {epicNum}-{storyNum}-{description})
		// Skip epic markers (epic-N) and retrospectives (epic-N-retrospective)
		parts := strings.SplitN(key, "-", 3)
		if len(parts) < 3 {
			continue
		}
		epicNum, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		storyNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		matches = append(matches, storySort{key: key, epicNum: epicNum, storyNum: storyNum})
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no stories found with status: %s", candidate)
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].epicNum != matches[j].epicNum {
			return matches[i].epicNum < matches[j].epicNum
		}
		return matches[i].storyNum < matches[j].storyNum
	})

	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = m.key
	}

	fmt.Printf("Resolved status %q to %d stories: %s\n", candidate, len(keys), strings.Join(keys, ", "))
	return keys, nil
}
