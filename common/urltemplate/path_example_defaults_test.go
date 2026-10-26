package urltemplate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The cap bounds how many distinct unmatched paths the shared cache holds per
// workload per direction. It is the only thing standing between a high-cardinality
// service and an unbounded Redis key, so a non-positive value has to become the
// default rather than being passed through as "store nothing" or "store it all".
func TestResolveMaxExamplePathsPerWorkload(t *testing.T) {
	for _, tc := range []struct {
		name  string
		given *int
		want  int
	}{
		{"unset", nil, DefaultMaxExamplePathsPerWorkload},
		{"zero", pxIntPtr(0), DefaultMaxExamplePathsPerWorkload},
		{"negative", pxIntPtr(-1), DefaultMaxExamplePathsPerWorkload},
		{"large negative", pxIntPtr(-5000), DefaultMaxExamplePathsPerWorkload},
		{"one", pxIntPtr(1), 1},
		{"below the default", pxIntPtr(10), 10},
		{"above the default", pxIntPtr(50000), 50000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveMaxExamplePathsPerWorkload(tc.given))
		})
	}

	// An explicit value equal to the default must still be read from the pointer
	// rather than short-circuited, so that "the user chose 5000" and "nobody chose
	// anything" cannot be told apart by the result but are both honoured.
	assert.Equal(t, DefaultMaxExamplePathsPerWorkload,
		ResolveMaxExamplePathsPerWorkload(pxIntPtr(DefaultMaxExamplePathsPerWorkload)))
}

// The idle TTL is a user-supplied string that Odigos never validates on the way
// in: helm, the remote config and the local UI overlay all accept any text. A
// typo therefore has to resolve to the default here, because this is the last
// place before the value becomes a cache expiry - and a zero or negative expiry
// would drop every path example as soon as it is written.
func TestResolvePathExampleIdleTTL(t *testing.T) {
	for _, tc := range []struct {
		name  string
		given string
		want  time.Duration
	}{
		{"empty", "", DefaultPathExampleIdleTTL},
		{"unparsable", "forever", DefaultPathExampleIdleTTL},
		{"missing unit", "48", DefaultPathExampleIdleTTL},
		{"days are not a Go duration unit", "2d", DefaultPathExampleIdleTTL},
		{"whitespace", " 48h ", DefaultPathExampleIdleTTL},
		{"zero", "0s", DefaultPathExampleIdleTTL},
		// "0" is the one input Go's parser accepts without a unit, so it is the
		// only row that reaches the non-positive check with a nil error.
		{"bare zero", "0", DefaultPathExampleIdleTTL},
		{"negative", "-1h", DefaultPathExampleIdleTTL},
		{"hours", "6h", 6 * time.Hour},
		{"minutes", "90m", 90 * time.Minute},
		{"seconds", "30s", 30 * time.Second},
		{"fractional", "0.5h", 30 * time.Minute},
		{"compound", "1h30m", 90 * time.Minute},
		{"longer than the default", "168h", 168 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolvePathExampleIdleTTL(tc.given))
		})
	}
}

// The two documented defaults are quoted verbatim in the helm values, the
// JSON schema, the GraphQL field docs and the gateway exporter's own fallback.
// Pinning the literals here makes a change to either one fail in this module
// first, where the other copies are a grep away.
func TestPathExampleDefaults(t *testing.T) {
	assert.Equal(t, 5000, DefaultMaxExamplePathsPerWorkload)
	assert.Equal(t, 48*time.Hour, DefaultPathExampleIdleTTL)
	assert.Positive(t, DefaultPathExampleIdleTTL,
		"a non-positive default would expire every path example on write")
	assert.Positive(t, DefaultMaxExamplePathsPerWorkload)
}

func pxIntPtr(v int) *int { return &v }
