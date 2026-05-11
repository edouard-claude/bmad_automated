package cli

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

// Styles for status display.
var (
	statusDoneStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusInProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	statusPendingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusEpicStyle       = lipgloss.NewStyle().Bold(true)
)

type epicGroup struct {
	id      int
	stories []storyEntry
}

type storyEntry struct {
	key    string
	num    int
	status status.Status
}

func newStatusCommand(app *App) *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "status [epic-id]",
		Short: "Display sprint status overview",
		Long: `Display the current sprint status, showing stories grouped by epic.

By default, only shows stories that are not yet done. Stories are grouped
by their epic number and sorted numerically.

Each story is displayed with a status icon:
  ` + "`✓`" + `  done
  ` + "`●`" + `  in-progress, review, ready-for-dev
  ` + "`○`" + `  backlog

Use --all to include completed stories. Pass an epic ID to filter
to a specific epic.

Examples:
  bmad-automate status              # All non-done stories, grouped by epic
  bmad-automate status --all        # Include done stories
  bmad-automate status 3            # Only epic 3`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintStatus, err := app.StatusReader.Read()
			if err != nil {
				cmd.SilenceUsage = true
				return fmt.Errorf("failed to read status: %w", err)
			}

			var epicFilter string
			if len(args) == 1 {
				epicFilter = args[0]
			}

			renderStatus(cmd.OutOrStdout(), sprintStatus.DevelopmentStatus, epicFilter, showAll)
			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Include done stories")

	return cmd
}

func renderStatus(out io.Writer, devStatus map[string]status.Status, epicFilter string, showAll bool) {
	epics := groupByEpic(devStatus)

	// Filter by epic if specified
	if epicFilter != "" {
		var filtered []epicGroup
		for _, e := range epics {
			if strconv.Itoa(e.id) == epicFilter {
				filtered = append(filtered, e)
			}
		}
		epics = filtered
	}

	// Sort epics numerically
	sort.Slice(epics, func(i, j int) bool {
		return epics[i].id < epics[j].id
	})

	// Count global stats across displayed epics
	counts := map[status.Status]int{}
	for _, epic := range epics {
		for _, s := range epic.stories {
			counts[s.status]++
		}
	}

	// Render each epic
	printed := false
	for _, epic := range epics {
		// Filter done stories unless --all
		var stories []storyEntry
		for _, s := range epic.stories {
			if !showAll && s.status == status.StatusDone {
				continue
			}
			stories = append(stories, s)
		}

		if len(stories) == 0 {
			continue
		}

		epicStatus := computeEpicStatus(epic.stories)

		if printed {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "%s (%s)\n", statusEpicStyle.Render(fmt.Sprintf("Epic %d", epic.id)), epicStatus)

		// Find max key length for alignment
		maxLen := 0
		for _, s := range stories {
			if len(s.key) > maxLen {
				maxLen = len(s.key)
			}
		}

		for _, s := range stories {
			icon := styledStatusIcon(s.status)
			fmt.Fprintf(out, "  %s %-*s  %s\n", icon, maxLen, s.key, s.status)
		}
		printed = true
	}

	// Summary line
	if printed {
		fmt.Fprintln(out)
	}
	renderSummary(out, counts)
}

func groupByEpic(devStatus map[string]status.Status) []epicGroup {
	epicMap := make(map[int]*epicGroup)

	for key, st := range devStatus {
		parts := strings.SplitN(key, "-", 3)
		if len(parts) < 2 {
			continue
		}

		epicID, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		storyNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		if _, ok := epicMap[epicID]; !ok {
			epicMap[epicID] = &epicGroup{id: epicID}
		}
		epicMap[epicID].stories = append(epicMap[epicID].stories, storyEntry{
			key:    key,
			num:    storyNum,
			status: st,
		})
	}

	// Sort stories within each epic
	result := make([]epicGroup, 0, len(epicMap))
	for _, epic := range epicMap {
		sort.Slice(epic.stories, func(i, j int) bool {
			return epic.stories[i].num < epic.stories[j].num
		})
		result = append(result, *epic)
	}

	return result
}

func computeEpicStatus(stories []storyEntry) string {
	allDone := true
	hasInProgress := false

	for _, s := range stories {
		if s.status != status.StatusDone {
			allDone = false
		}
		if s.status == status.StatusInProgress || s.status == status.StatusReview {
			hasInProgress = true
		}
	}

	if allDone {
		return string(status.StatusDone)
	}
	if hasInProgress {
		return string(status.StatusInProgress)
	}
	return string(status.StatusBacklog)
}

func styledStatusIcon(s status.Status) string {
	switch s {
	case status.StatusDone:
		return statusDoneStyle.Render("✓")
	case status.StatusInProgress, status.StatusReview, status.StatusReadyForDev:
		return statusInProgressStyle.Render("●")
	default:
		return statusPendingStyle.Render("○")
	}
}

func renderSummary(out io.Writer, counts map[status.Status]int) {
	order := []status.Status{
		status.StatusInProgress,
		status.StatusReview,
		status.StatusReadyForDev,
		status.StatusBacklog,
		status.StatusDone,
	}

	var parts []string
	for _, s := range order {
		if c, ok := counts[s]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, s))
		}
	}
	fmt.Fprintf(out, "Summary: %s\n", strings.Join(parts, ", "))
}
