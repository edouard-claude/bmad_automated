package router

import (
	"errors"
	"testing"

	"github.com/edouard-claude/bmad_automated/internal/status"
)

func TestGetWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		status         status.Status
		wantWorkflow   string
		wantErr        error
		wantErrMessage string
	}{
		{
			name:         "backlog status returns create-story workflow",
			status:       status.StatusBacklog,
			wantWorkflow: "create-story",
			wantErr:      nil,
		},
		{
			name:         "ready-for-dev status returns dev-story workflow",
			status:       status.StatusReadyForDev,
			wantWorkflow: "dev-story",
			wantErr:      nil,
		},
		{
			name:         "in-progress status returns dev-story workflow",
			status:       status.StatusInProgress,
			wantWorkflow: "dev-story",
			wantErr:      nil,
		},
		{
			name:         "review status returns code-review workflow",
			status:       status.StatusReview,
			wantWorkflow: "code-review",
			wantErr:      nil,
		},
		{
			name:         "done status returns ErrStoryComplete",
			status:       status.StatusDone,
			wantWorkflow: "",
			wantErr:      ErrStoryComplete,
		},
		{
			name:         "unknown status returns ErrUnknownStatus",
			status:       status.Status("invalid"),
			wantWorkflow: "",
			wantErr:      ErrUnknownStatus,
		},
		{
			name:         "empty status returns ErrUnknownStatus",
			status:       status.Status(""),
			wantWorkflow: "",
			wantErr:      ErrUnknownStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWorkflow, gotErr := GetWorkflow(tt.status)

			if gotWorkflow != tt.wantWorkflow {
				t.Errorf("GetWorkflow(%q) workflow = %q, want %q", tt.status, gotWorkflow, tt.wantWorkflow)
			}

			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("GetWorkflow(%q) err = nil, want %v", tt.status, tt.wantErr)
				} else if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("GetWorkflow(%q) err = %v, want %v", tt.status, gotErr, tt.wantErr)
				}
			} else if gotErr != nil {
				t.Errorf("GetWorkflow(%q) err = %v, want nil", tt.status, gotErr)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Run("ErrStoryComplete has descriptive message", func(t *testing.T) {
		if ErrStoryComplete.Error() == "" {
			t.Error("ErrStoryComplete should have a non-empty error message")
		}
	})

	t.Run("ErrUnknownStatus has descriptive message", func(t *testing.T) {
		if ErrUnknownStatus.Error() == "" {
			t.Error("ErrUnknownStatus should have a non-empty error message")
		}
	})

	t.Run("sentinel errors are distinct", func(t *testing.T) {
		if errors.Is(ErrStoryComplete, ErrUnknownStatus) {
			t.Error("ErrStoryComplete and ErrUnknownStatus should be distinct errors")
		}
	})
}
