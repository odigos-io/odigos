package nodecollectorsgroup

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

// TestCalculateMemoryLimiterHardLimitMiB verifies the memory_limiter hard limit formula
// max(limit-50, limit*85%). The formula is designed so that:
//   - small container limits (below ~333MiB) use the 85% ratio — avoids the old bug where
//     a fixed 50MiB offset ate most of the budget (128MiB → 78MiB hard limit = 61%).
//   - larger container limits (>=~333MiB) keep the fixed 50MiB headroom that the OTel
//     memory_limiter docs recommend.
func TestCalculateMemoryLimiterHardLimitMiB(t *testing.T) {
	tests := []struct {
		name          string
		containerMiB  int
		wantHardLimit int
	}{
		// small containers — 85% ratio wins
		{"64MiB (tiny)", 64, 54},    // max(14, 54) = 54
		{"128MiB", 128, 108},        // max(78, 108) = 108  (was 78 pre-fix)
		{"192MiB", 192, 163},        // max(142, 163) = 163
		{"256MiB", 256, 217},        // max(206, 217) = 217  (was 206 pre-fix)
		{"320MiB", 320, 272},        // max(270, 272) = 272

		// crossover region — around 333MiB the two strategies meet
		{"333MiB", 333, 283},        // max(283, 283) = 283
		{"334MiB", 334, 284},        // max(284, 283) = 284 (fixed offset wins by 1)

		// larger containers — fixed -50 headroom wins (unchanged from pre-fix behavior)
		{"384MiB", 384, 334},        // max(334, 326) = 334
		{"512MiB (odigos default limit)", 512, 462}, // max(462, 435) = 462
		{"768MiB", 768, 718},        // max(718, 652) = 718
		{"1024MiB", 1024, 974},      // max(974, 870) = 974
		{"2048MiB", 2048, 1998},     // max(1998, 1740) = 1998
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMemoryLimiterHardLimitMiB(tt.containerMiB)
			if got != tt.wantHardLimit {
				t.Errorf("calculateMemoryLimiterHardLimitMiB(%d) = %d, want %d",
					tt.containerMiB, got, tt.wantHardLimit)
			}
			// sanity: hard limit must always leave at least min(50, 15%) headroom —
			// that is exactly what the max(limit-50, limit*85%) formula guarantees.
			headroom := tt.containerMiB - got
			wantMinHeadroom := tt.containerMiB * 15 / 100 // 15% rounded down
			if wantMinHeadroom > defaultMemoryLimiterLimitDiffMib {
				wantMinHeadroom = defaultMemoryLimiterLimitDiffMib
			}
			if headroom < wantMinHeadroom {
				t.Errorf("hard limit %d leaves only %d MiB headroom in %d MiB container, want at least %d",
					got, headroom, tt.containerMiB, wantMinHeadroom)
			}
		})
	}
}

// TestGetResourceSettings_NodeCollector_Defaults verifies that with no user override the
// node collector gets the Odigos defaults (256 request / 512 limit) and the ratios are
// computed correctly.
func TestGetResourceSettings_NodeCollector_Defaults(t *testing.T) {
	got := getResourceSettings(common.OdigosConfiguration{})

	checkInt(t, "MemoryRequestMiB", got.MemoryRequestMiB, 256)
	checkInt(t, "MemoryLimitMiB", got.MemoryLimitMiB, 512)
	// 512MiB is above the crossover, so the fixed -50 wins: 462.
	checkInt(t, "MemoryLimiterLimitMiB", got.MemoryLimiterLimitMiB, 462)
	// spike is 20% of hard: 462 * 0.2 = 92.
	checkInt(t, "MemoryLimiterSpikeLimitMiB", got.MemoryLimiterSpikeLimitMiB, 92)
	// gomemlimit is 80% of hard: 462 * 0.8 = 369.
	checkInt(t, "GomemlimitMiB", got.GomemlimitMiB, 369)
	checkInt(t, "CpuRequestMillicores", got.CpuRequestMillicores, 250)
	checkInt(t, "CpuLimitMillicores", got.CpuLimitMillicores, 500)
}

// TestGetResourceSettings_NodeCollector_Sizes verifies the full chain of derived values
// for several container sizes. This is the regression test for the customer scenario
// where requestMemoryMiB=64 / limitMemoryMiB=128 was producing gomemlimit=62 (48% of
// container), leaving no budget for the Go runtime baseline.
func TestGetResourceSettings_NodeCollector_Sizes(t *testing.T) {
	tests := []struct {
		name              string
		requestMiB        int
		limitMiB          int
		wantHardLimit     int
		wantSpikeLimit    int
		wantGomemlimit    int
	}{
		{
			// the customer case — tiny container, previously produced 78/15/62.
			name:           "64/128 (small override)",
			requestMiB:     64,
			limitMiB:       128,
			wantHardLimit:  108, // was 78 pre-fix (61% of container)
			wantSpikeLimit: 21,  // 108 * 0.20 (was 15)
			wantGomemlimit: 86,  // 108 * 0.80 (was 62 — 48% of container)
		},
		{
			// default-ish Odigos sizing.
			name:           "128/256",
			requestMiB:     128,
			limitMiB:       256,
			wantHardLimit:  217, // was 206 pre-fix
			wantSpikeLimit: 43,  // 217 * 0.20
			wantGomemlimit: 173, // 217 * 0.80
		},
		{
			// Odigos defaults — should be unchanged from pre-fix.
			name:           "256/512 (defaults)",
			requestMiB:     256,
			limitMiB:       512,
			wantHardLimit:  462,
			wantSpikeLimit: 92, // 462 * 0.20 = 92.4 → 92
			wantGomemlimit: 369,
		},
		{
			// Large container — unchanged from pre-fix.
			name:           "512/1024",
			requestMiB:     512,
			limitMiB:       1024,
			wantHardLimit:  974,
			wantSpikeLimit: 194, // 974 * 0.20 = 194.8 → 194
			wantGomemlimit: 779, // 974 * 0.80 = 779.2 → 779
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := common.OdigosConfiguration{
				CollectorNode: &common.CollectorNodeConfiguration{
					RequestMemoryMiB: tt.requestMiB,
					LimitMemoryMiB:   tt.limitMiB,
				},
			}
			got := getResourceSettings(cfg)

			checkInt(t, "MemoryRequestMiB", got.MemoryRequestMiB, tt.requestMiB)
			checkInt(t, "MemoryLimitMiB", got.MemoryLimitMiB, tt.limitMiB)
			checkInt(t, "MemoryLimiterLimitMiB", got.MemoryLimiterLimitMiB, tt.wantHardLimit)
			checkInt(t, "MemoryLimiterSpikeLimitMiB", got.MemoryLimiterSpikeLimitMiB, tt.wantSpikeLimit)
			checkInt(t, "GomemlimitMiB", got.GomemlimitMiB, tt.wantGomemlimit)

			// Invariants:
			// - hard limit must be strictly below container limit (otherwise the limiter
			//   fires at the same moment the kernel OOMKills).
			if got.MemoryLimiterLimitMiB >= got.MemoryLimitMiB {
				t.Errorf("hard limit (%d) must be < container limit (%d)",
					got.MemoryLimiterLimitMiB, got.MemoryLimitMiB)
			}
			// - gomemlimit must be below hard limit (Go GC reacts before the limiter trips).
			if got.GomemlimitMiB >= got.MemoryLimiterLimitMiB {
				t.Errorf("gomemlimit (%d) must be < hard limit (%d)",
					got.GomemlimitMiB, got.MemoryLimiterLimitMiB)
			}
			// - hard limit must leave at least min(50, 15%) headroom in the container,
			//   matching the max(limit-50, limit*85%) formula.
			headroom := got.MemoryLimitMiB - got.MemoryLimiterLimitMiB
			wantMinHeadroom := got.MemoryLimitMiB * 15 / 100
			if wantMinHeadroom > 50 {
				wantMinHeadroom = 50
			}
			if headroom < wantMinHeadroom {
				t.Errorf("hard limit %d leaves only %d MiB headroom in %d MiB container, want at least %d",
					got.MemoryLimiterLimitMiB, headroom, got.MemoryLimitMiB, wantMinHeadroom)
			}
		})
	}
}

// TestGetResourceSettings_NodeCollector_UserOverrides verifies that explicit user-provided
// values take precedence over the computed defaults (i.e. we never recompute on top of a
// user override).
func TestGetResourceSettings_NodeCollector_UserOverrides(t *testing.T) {
	cfg := common.OdigosConfiguration{
		CollectorNode: &common.CollectorNodeConfiguration{
			RequestMemoryMiB:           300,
			LimitMemoryMiB:             600,
			MemoryLimiterLimitMiB:      500,
			MemoryLimiterSpikeLimitMiB: 77,
			GoMemLimitMib:              400,
			RequestCPUm:                111,
			LimitCPUm:                  222,
		},
	}
	got := getResourceSettings(cfg)

	checkInt(t, "MemoryRequestMiB", got.MemoryRequestMiB, 300)
	checkInt(t, "MemoryLimitMiB", got.MemoryLimitMiB, 600)
	checkInt(t, "MemoryLimiterLimitMiB", got.MemoryLimiterLimitMiB, 500)
	checkInt(t, "MemoryLimiterSpikeLimitMiB", got.MemoryLimiterSpikeLimitMiB, 77)
	checkInt(t, "GomemlimitMiB", got.GomemlimitMiB, 400)
	checkInt(t, "CpuRequestMillicores", got.CpuRequestMillicores, 111)
	checkInt(t, "CpuLimitMillicores", got.CpuLimitMillicores, 222)
}

func checkInt(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}
