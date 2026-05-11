package status_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

// This example demonstrates using Reader to read sprint status from YAML files.
// The reader queries story statuses and retrieves epic story lists.
func Example_reader() {
	// Create a temporary directory with a sample sprint-status.yaml
	tmpDir, err := os.MkdirTemp("", "status-reader")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Create the status file path structure
	statusDir := filepath.Join(tmpDir, "_bmad-output", "implementation-artifacts")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Write a sample sprint-status.yaml
	statusYAML := `development_status:
  7-1-define-schema: backlog
  7-2-add-api: ready-for-dev
  7-3-add-tests: done
`
	statusFile := filepath.Join(statusDir, "sprint-status.yaml")
	if err := os.WriteFile(statusFile, []byte(statusYAML), 0644); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a reader with the temp directory as base path
	reader := status.NewReader(tmpDir)

	// GetStoryStatus returns the status for a specific story
	s, err := reader.GetStoryStatus("7-1-define-schema")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Story 7-1 status:", s)

	// Read returns the full sprint status structure
	sprint, err := reader.Read()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Total stories:", len(sprint.DevelopmentStatus))
	// Output:
	// Story 7-1 status: backlog
	// Total stories: 3
}

// This example demonstrates using Writer to update story statuses in YAML files.
// The writer preserves formatting and uses atomic writes for safety.
func Example_writer() {
	// Create a temporary directory with a sample sprint-status.yaml
	tmpDir, err := os.MkdirTemp("", "status-writer")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Create the status file path structure
	statusDir := filepath.Join(tmpDir, "_bmad-output", "implementation-artifacts")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Write a sample sprint-status.yaml
	statusYAML := `development_status:
  7-1-define-schema: backlog
  7-2-add-api: ready-for-dev
`
	statusFile := filepath.Join(statusDir, "sprint-status.yaml")
	if err := os.WriteFile(statusFile, []byte(statusYAML), 0644); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Read initial status
	reader := status.NewReader(tmpDir)
	initial, _ := reader.GetStoryStatus("7-1-define-schema")
	fmt.Println("Initial status:", initial)

	// Create a writer and update the status
	writer := status.NewWriter(tmpDir)
	if err := writer.UpdateStatus("7-1-define-schema", status.StatusInProgress); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Verify the update
	updated, _ := reader.GetStoryStatus("7-1-define-schema")
	fmt.Println("Updated status:", updated)

	// Invalid status values are rejected
	err = writer.UpdateStatus("7-1-define-schema", status.Status("invalid"))
	fmt.Println("Invalid status rejected:", err != nil)
	// Output:
	// Initial status: backlog
	// Updated status: in-progress
	// Invalid status rejected: true
}
