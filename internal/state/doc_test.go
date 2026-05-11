package state_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edouard-claude/bmad_automated/internal/state"
)

// This example demonstrates the Manager interface for state persistence.
// The manager handles save, load, and clear operations for lifecycle state.
func Example_manager() {
	// Create a temporary directory for the example
	tmpDir, err := os.MkdirTemp("", "state-example")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Create a manager for the temporary directory
	mgr := state.NewManager(tmpDir)

	// Initially, no state exists
	fmt.Println("State exists initially:", mgr.Exists())

	// Save some state (typically done when execution fails)
	s := state.State{
		StoryKey:    "7-1-define-schema",
		StepIndex:   2,
		TotalSteps:  4,
		StartStatus: "backlog",
	}
	if err := mgr.Save(s); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("State exists after save:", mgr.Exists())

	// Load the state back (typically done when resuming)
	loaded, err := mgr.Load()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Loaded story key:", loaded.StoryKey)
	fmt.Println("Loaded step index:", loaded.StepIndex)

	// Clear the state (typically done after successful completion)
	if err := mgr.Clear(); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("State exists after clear:", mgr.Exists())
	// Output:
	// State exists initially: false
	// State exists after save: true
	// Loaded story key: 7-1-define-schema
	// Loaded step index: 2
	// State exists after clear: false
}

// This example demonstrates resume state detection and loading.
// The ErrNoState sentinel indicates a fresh execution should start.
func Example_resumeState() {
	// Create a temporary directory for the example
	tmpDir, err := os.MkdirTemp("", "resume-example")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	mgr := state.NewManager(tmpDir)

	// When no state file exists, Load returns ErrNoState
	_, err = mgr.Load()
	fmt.Println("No state file returns ErrNoState:", errors.Is(err, state.ErrNoState))

	// Create a state file manually to simulate a failed execution
	stateFile := filepath.Join(tmpDir, state.StateFileName)
	stateJSON := `{"story_key":"7-2-add-api","step_index":1,"total_steps":3,"start_status":"ready-for-dev"}`
	if err := os.WriteFile(stateFile, []byte(stateJSON), 0644); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Now Load returns the saved state for resume
	loaded, err := mgr.Load()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Resume story:", loaded.StoryKey)
	fmt.Println("Resume from step:", loaded.StepIndex, "of", loaded.TotalSteps)
	// Output:
	// No state file returns ErrNoState: true
	// Resume story: 7-2-add-api
	// Resume from step: 1 of 3
}
