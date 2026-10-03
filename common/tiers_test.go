package common

import "testing"

func TestOdigosTierIsEnterprise(t *testing.T) {
	testCases := []struct {
		name     string
		tier     OdigosTier
		expected bool
	}{
		{"Community", CommunityOdigosTier, false},
		{"Cloud", CloudOdigosTier, true},
		{"OnPrem", OnPremOdigosTier, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tier.IsEnterprise(); got != tc.expected {
				t.Errorf("OdigosTier(%q).IsEnterprise() = %v, want %v", tc.tier, got, tc.expected)
			}
		})
	}
}

// IsEnterprise treats anything that is not community as enterprise, so an unrecognized tier
// would enable enterprise-only components in an OSS collector and crash it at config load.
// Callers must normalize the tier before calling (see env.GetOdigosTierFromEnv), and this test
// pins the direction of the check so the normalization requirement is not silently dropped.
func TestOdigosTierIsEnterprise_UnrecognizedTierIsNotCommunity(t *testing.T) {
	for _, tier := range []OdigosTier{OdigosTier(""), OdigosTier("Community"), OdigosTier("bogus")} {
		if !tier.IsEnterprise() {
			t.Errorf("OdigosTier(%q).IsEnterprise() = false, want true: only the exact %q value is community",
				tier, CommunityOdigosTier)
		}
	}
}
