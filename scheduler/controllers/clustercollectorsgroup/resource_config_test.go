package clustercollectorsgroup

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

// TestCalculateMemoryLimiterHardLimitMiB_Gateway verifies the gateway uses the same
// max(base-50, base*85%) formula as the node collector, so small base sizes get a
// percentage-based floor instead of the fixed 50MiB offset eating the budget.
func TestCalculateMemoryLimiterHardLimitMiB_Gateway(t *testing.T) {
	tests := []struct {
		name     string
		baseMiB  int
		wantHard int
	}{
		{"64MiB", 64, 54},     // max(14, 54) = 54
		{"128MiB", 128, 108},  // max(78, 108) = 108
		{"256MiB", 256, 217},  // max(206, 217) = 217
		{"333MiB (crossover)", 333, 283},
		{"334MiB", 334, 284},
		{"500MiB (default request)", 500, 450}, // max(450, 425) = 450 (unchanged)
		{"1000MiB", 1000, 950}, // max(950, 850) = 950 (unchanged)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateMemoryLimiterHardLimitMiB(tt.baseMiB); got != tt.wantHard {
				t.Errorf("calculateMemoryLimiterHardLimitMiB(%d) = %d, want %d", tt.baseMiB, got, tt.wantHard)
			}
		})
	}
}

// TestGetGatewayResourceSettings_Defaults verifies the default gateway sizing —
// should be unchanged by the ratio fix since the default request (500MiB) is well
// above the ~333MiB crossover.
func TestGetGatewayResourceSettings_Defaults(t *testing.T) {
	got := getGatewayResourceSettings(&common.OdigosConfiguration{})

	if got.MemoryRequestMiB != 500 {
		t.Errorf("MemoryRequestMiB = %d, want 500", got.MemoryRequestMiB)
	}
	// hard limit = 500 - 50 = 450 (fixed offset wins above crossover).
	if got.MemoryLimiterLimitMiB != 450 {
		t.Errorf("MemoryLimiterLimitMiB = %d, want 450", got.MemoryLimiterLimitMiB)
	}
	// spike = 450 * 0.20 = 90.
	if got.MemoryLimiterSpikeLimitMiB != 90 {
		t.Errorf("MemoryLimiterSpikeLimitMiB = %d, want 90", got.MemoryLimiterSpikeLimitMiB)
	}
	// gomemlimit = 450 * 0.80 = 360.
	if got.GomemlimitMiB != 360 {
		t.Errorf("GomemlimitMiB = %d, want 360", got.GomemlimitMiB)
	}
}

// TestGetGatewayResourceSettings_SmallOverride verifies the fix kicks in when a user
// downsizes the gateway below the ~333MiB crossover.
func TestGetGatewayResourceSettings_SmallOverride(t *testing.T) {
	cfg := &common.OdigosConfiguration{
		CollectorGateway: &common.CollectorGatewayConfiguration{
			RequestMemoryMiB: 128,
		},
	}
	got := getGatewayResourceSettings(cfg)

	if got.MemoryRequestMiB != 128 {
		t.Errorf("MemoryRequestMiB = %d, want 128", got.MemoryRequestMiB)
	}
	// Pre-fix: 128 - 50 = 78. Post-fix: max(78, int(128*0.85)) = 108.
	if got.MemoryLimiterLimitMiB != 108 {
		t.Errorf("MemoryLimiterLimitMiB = %d, want 108 (post-fix), was 78 pre-fix", got.MemoryLimiterLimitMiB)
	}
	if got.MemoryLimiterSpikeLimitMiB != 21 { // 108 * 0.20 = 21.6 → 21
		t.Errorf("MemoryLimiterSpikeLimitMiB = %d, want 21", got.MemoryLimiterSpikeLimitMiB)
	}
	if got.GomemlimitMiB != 86 { // 108 * 0.80 = 86.4 → 86
		t.Errorf("GomemlimitMiB = %d, want 86", got.GomemlimitMiB)
	}
}
