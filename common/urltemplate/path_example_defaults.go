package urltemplate

import "time"

const (
	// DefaultMaxExamplePathsPerWorkload caps distinct unmatched HTTP paths
	// stored per workload for live-traffic learning when
	// MaxExamplePathsPerWorkload is unset or ≤ 0.
	DefaultMaxExamplePathsPerWorkload = 5000

	// DefaultPathExampleIdleTTL is the sliding idle window for an unmatched
	// path example in cacheDb. After this long without a new observation the
	// field expires so another path can take the quota slot. Used when
	// PathExampleIdleTTL is unset or invalid.
	DefaultPathExampleIdleTTL = 48 * time.Hour
)

// ResolveMaxExamplePathsPerWorkload returns maxPaths when set and positive,
// else DefaultMaxExamplePathsPerWorkload.
func ResolveMaxExamplePathsPerWorkload(maxPaths *int) int {
	if maxPaths == nil || *maxPaths <= 0 {
		return DefaultMaxExamplePathsPerWorkload
	}
	return *maxPaths
}

// ResolvePathExampleIdleTTL parses a Go duration (e.g. "48h") and returns it
// when positive, else DefaultPathExampleIdleTTL.
func ResolvePathExampleIdleTTL(idleTTL string) time.Duration {
	if idleTTL == "" {
		return DefaultPathExampleIdleTTL
	}
	d, err := time.ParseDuration(idleTTL)
	if err != nil || d <= 0 {
		return DefaultPathExampleIdleTTL
	}
	return d
}
