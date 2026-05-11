package router

import (
	"errors"
	"testing"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

func TestGetLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		status    status.Status
		wantSteps []LifecycleStep
		wantErr   error
	}{
		{
			name:   "backlog status returns full lifecycle",
			status: status.StatusBacklog,
			wantSteps: []LifecycleStep{
				{Workflow: "create-story", NextStatus: status.StatusReadyForDev},
				{Workflow: "dev-story", NextStatus: status.StatusReview},
				{Workflow: "code-review", NextStatus: status.StatusDone},
				{Workflow: "git-commit", NextStatus: status.StatusDone},
			},
			wantErr: nil,
		},
		{
			name:   "ready-for-dev status returns dev through commit",
			status: status.StatusReadyForDev,
			wantSteps: []LifecycleStep{
				{Workflow: "dev-story", NextStatus: status.StatusReview},
				{Workflow: "code-review", NextStatus: status.StatusDone},
				{Workflow: "git-commit", NextStatus: status.StatusDone},
			},
			wantErr: nil,
		},
		{
			name:   "in-progress status returns dev through commit",
			status: status.StatusInProgress,
			wantSteps: []LifecycleStep{
				{Workflow: "dev-story", NextStatus: status.StatusReview},
				{Workflow: "code-review", NextStatus: status.StatusDone},
				{Workflow: "git-commit", NextStatus: status.StatusDone},
			},
			wantErr: nil,
		},
		{
			name:   "review status returns review through commit",
			status: status.StatusReview,
			wantSteps: []LifecycleStep{
				{Workflow: "code-review", NextStatus: status.StatusDone},
				{Workflow: "git-commit", NextStatus: status.StatusDone},
			},
			wantErr: nil,
		},
		{
			name:      "done status returns ErrStoryComplete",
			status:    status.StatusDone,
			wantSteps: nil,
			wantErr:   ErrStoryComplete,
		},
		{
			name:      "unknown status returns ErrUnknownStatus",
			status:    status.Status("invalid"),
			wantSteps: nil,
			wantErr:   ErrUnknownStatus,
		},
		{
			name:      "empty status returns ErrUnknownStatus",
			status:    status.Status(""),
			wantSteps: nil,
			wantErr:   ErrUnknownStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSteps, gotErr := GetLifecycle(tt.status)

			// Check error
			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("GetLifecycle(%q) err = nil, want %v", tt.status, tt.wantErr)
				} else if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("GetLifecycle(%q) err = %v, want %v", tt.status, gotErr, tt.wantErr)
				}
				return
			}
			if gotErr != nil {
				t.Errorf("GetLifecycle(%q) err = %v, want nil", tt.status, gotErr)
				return
			}

			// Check steps count
			if len(gotSteps) != len(tt.wantSteps) {
				t.Errorf("GetLifecycle(%q) returned %d steps, want %d", tt.status, len(gotSteps), len(tt.wantSteps))
				return
			}

			// Check each step
			for i, wantStep := range tt.wantSteps {
				gotStep := gotSteps[i]
				if gotStep.Workflow != wantStep.Workflow {
					t.Errorf("GetLifecycle(%q) step[%d].Workflow = %q, want %q", tt.status, i, gotStep.Workflow, wantStep.Workflow)
				}
				if gotStep.NextStatus != wantStep.NextStatus {
					t.Errorf("GetLifecycle(%q) step[%d].NextStatus = %q, want %q", tt.status, i, gotStep.NextStatus, wantStep.NextStatus)
				}
			}
		})
	}
}
