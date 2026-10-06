package loaders

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

func loadersWithConfig(cfg *common.OdigosConfiguration) *Loaders {
	l := NewLoaders(logr.Discard(), nil)
	l.odigosConfiguration = cfg
	return l
}

// The rollback durations are free text in the odigos configuration: the settings UI
// writes them without validation and helm/GitOps users set them directly. Both call
// sites of GetAutoRollbackConfig dereference the result for every instrumented
// workload, so an unparsable duration has to fall back to the default instead of
// dropping the whole config.
func TestGetAutoRollbackConfig_InvalidDurations(t *testing.T) {
	tests := []struct {
		name                string
		graceTime           string
		stabilityWindow     string
		wantGraceTime       time.Duration
		wantStabilityWindow time.Duration
	}{
		{
			name:                "no durations set",
			wantGraceTime:       consts.DefaultAutoRollbackGraceTime,
			wantStabilityWindow: consts.DefaultAutoRollbackStabilityWindow,
		},
		{
			name:                "both valid",
			graceTime:           "30s",
			stabilityWindow:     "10m",
			wantGraceTime:       30 * time.Second,
			wantStabilityWindow: 10 * time.Minute,
		},
		{
			// a bare number is what the "time" settings component makes easy to type
			name:                "grace time missing a unit",
			graceTime:           "5",
			stabilityWindow:     "10m",
			wantGraceTime:       consts.DefaultAutoRollbackGraceTime,
			wantStabilityWindow: 10 * time.Minute,
		},
		{
			name:                "stability window in prose",
			graceTime:           "30s",
			stabilityWindow:     "5 minutes",
			wantGraceTime:       30 * time.Second,
			wantStabilityWindow: consts.DefaultAutoRollbackStabilityWindow,
		},
		{
			name:                "both invalid",
			graceTime:           "1h30",
			stabilityWindow:     "30 sec",
			wantGraceTime:       consts.DefaultAutoRollbackGraceTime,
			wantStabilityWindow: consts.DefaultAutoRollbackStabilityWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := loadersWithConfig(&common.OdigosConfiguration{
				RollbackGraceTime:       tt.graceTime,
				RollbackStabilityWindow: tt.stabilityWindow,
			})

			got := l.GetAutoRollbackConfig()
			if got == nil {
				t.Fatal("expected an auto rollback config, got nil")
			}
			if !got.Enabled {
				t.Fatal("expected auto rollback to be enabled when rollbackDisabled is unset")
			}
			if got.GraceTime != tt.wantGraceTime {
				t.Fatalf("expected grace time %s, got %s", tt.wantGraceTime, got.GraceTime)
			}
			if got.StabilityWindow != tt.wantStabilityWindow {
				t.Fatalf("expected stability window %s, got %s", tt.wantStabilityWindow, got.StabilityWindow)
			}
		})
	}
}

// An invalid duration must not silently re-enable auto rollback for a user who
// disabled it.
func TestGetAutoRollbackConfig_DisabledWithInvalidDuration(t *testing.T) {
	disabled := true
	l := loadersWithConfig(&common.OdigosConfiguration{
		RollbackDisabled:  &disabled,
		RollbackGraceTime: "not-a-duration",
	})

	got := l.GetAutoRollbackConfig()
	if got == nil {
		t.Fatal("expected an auto rollback config, got nil")
	}
	if got.Enabled {
		t.Fatal("expected auto rollback to stay disabled")
	}
}

func TestGetAutoRollbackConfig_NoOdigosConfiguration(t *testing.T) {
	if got := loadersWithConfig(nil).GetAutoRollbackConfig(); got != nil {
		t.Fatalf("expected nil without an odigos configuration, got %+v", got)
	}
}
