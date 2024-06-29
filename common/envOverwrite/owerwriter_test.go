package envOverwrite

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestGetPatchedEnvValue(t *testing.T) {
	nodeOptionsNativeCommunity, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
	nodeOptionsEbpfEnterprise, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkEbpfEnterprise)
	userVal := "--max-old-space-size=4096"

	// test different cases
	tests := []struct {
		name                 string
		envName              string
		observedValue        string
		sdk                  common.OtelSdk
		patchedValueExpected string
	}{
		{
			name:                 "un-relevant env var",
			envName:              "PATH",
			observedValue:        "/usr/local/bin:/usr/bin:/bin",
			sdk:                  common.OtelSdkNativeCommunity,
			patchedValueExpected: "",
		},
		{
			name:                 "only user value",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal,
			sdk:                  common.OtelSdkNativeCommunity,
			patchedValueExpected: userVal + " " + nodeOptionsNativeCommunity,
		},
		{
			name:                 "only odigos value",
			envName:              "NODE_OPTIONS",
			observedValue:        nodeOptionsNativeCommunity,
			sdk:                  common.OtelSdkNativeCommunity,
			patchedValueExpected: "",
		},
		{
			name:                 "user value with odigos value matching SDKs",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal + " " + nodeOptionsNativeCommunity,
			sdk:                  common.OtelSdkNativeCommunity,
			patchedValueExpected: userVal + " " + nodeOptionsNativeCommunity,
		},
		{
			name:                 "user value with odigos value with different SDKs",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal + " " + nodeOptionsNativeCommunity,
			sdk:                  common.OtelSdkEbpfEnterprise,
			patchedValueExpected: userVal + " " + nodeOptionsEbpfEnterprise,
		},
		{
			// No user values are observed, hence there is not need to patch
			// even if the observed value is different from the SDK value
			name:                 "observed odigos value different from SDK",
			envName:              "NODE_OPTIONS",
			observedValue:        nodeOptionsNativeCommunity,
			sdk:                  common.OtelSdkEbpfEnterprise,
			patchedValueExpected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchedValue := GetPatchedEnvValue(tt.envName, tt.observedValue, tt.sdk)
			if patchedValue == nil {
				assert.Equal(t, tt.patchedValueExpected, "", "mismatch in GetPatchedEnvValue: %s", tt.name)
			} else {
				assert.Equal(t, tt.patchedValueExpected, *patchedValue, "mismatch in GetPatchedEnvValue: %s", tt.name)
			}
		})
	}

}

// func TestShouldPatch(t *testing.T) {
// 	nodeOptionsNativeCommunity, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	nodeOptionsEbpfEnterprise, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkEbpfEnterprise)
// 	userVal := "--max-old-space-size=4096"

// 	// test different cases
// 	tests := []struct {
// 		name                 string
// 		envName              string
// 		observedValue        string
// 		sdk                  common.OtelSdk
// 		shouldPatchExpected  bool
// 		patchedValueExpected string
// 	}{
// 		{
// 			name:                "un-relevant env var",
// 			envName:             "PATH",
// 			observedValue:       "/usr/local/bin:/usr/bin:/bin",
// 			sdk:                 common.OtelSdkNativeCommunity,
// 			shouldPatchExpected: false,
// 		},
// 		{
// 			name:                 "only user value",
// 			envName:              "NODE_OPTIONS",
// 			observedValue:        userVal,
// 			sdk:                  common.OtelSdkNativeCommunity,
// 			shouldPatchExpected:  true,
// 			patchedValueExpected: userVal + " " + nodeOptionsNativeCommunity,
// 		},
// 		{
// 			name:                "only odigos value",
// 			envName:             "NODE_OPTIONS",
// 			observedValue:       nodeOptionsNativeCommunity,
// 			sdk:                 common.OtelSdkNativeCommunity,
// 			shouldPatchExpected: false,
// 		},
// 		{
// 			name:                "user value with odigos value matching SDKs",
// 			envName:             "NODE_OPTIONS",
// 			observedValue:       userVal + " " + nodeOptionsNativeCommunity,
// 			sdk:                 common.OtelSdkNativeCommunity,
// 			shouldPatchExpected: false,
// 		},
// 		{
// 			name:                 "user value with odigos value with different SDKs",
// 			envName:              "NODE_OPTIONS",
// 			observedValue:        userVal + " " + nodeOptionsNativeCommunity,
// 			sdk:                  common.OtelSdkEbpfEnterprise,
// 			shouldPatchExpected:  true,
// 			patchedValueExpected: userVal + " " + nodeOptionsEbpfEnterprise,
// 		},
// 		{
// 			// No user values are observed, hence there is not need to patch
// 			// even if the observed value is different from the SDK value
// 			name:                "observed odigos value different from SDK",
// 			envName:             "NODE_OPTIONS",
// 			observedValue:       nodeOptionsNativeCommunity,
// 			sdk:                 common.OtelSdkEbpfEnterprise,
// 			shouldPatchExpected: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gotShouldPatch := ShouldPatch(tt.envName, tt.observedValue, tt.sdk)
// 			assert.Equal(t, tt.shouldPatchExpected, gotShouldPatch, "mismatch in ShouldPatch: %s", tt.name)
// 			if gotShouldPatch {
// 				gotPatchedValue := Patch(tt.envName, tt.observedValue, tt.sdk)
// 				assert.Equal(t, tt.patchedValueExpected, gotPatchedValue)
// 			}
// 		})
// 	}
// }

// func TestShouldRevert(t *testing.T) {
// 	nodeOptionsNativeCommunity, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
// 	userVal := "--max-old-space-size=4096"

// 	// test different cases
// 	tests := []struct {
// 		name          string
// 		envName       string
// 		observedValue string
// 		wantRevert    bool
// 	}{
// 		{
// 			name:          "un-relevant env var",
// 			envName:       "PATH",
// 			observedValue: "/usr/local/bin:/usr/bin:/bin",
// 			wantRevert:    false,
// 		},
// 		{
// 			name:          "only user value",
// 			envName:       "NODE_OPTIONS",
// 			observedValue: userVal,
// 			wantRevert:    false,
// 		},
// 		{
// 			name:          "only odigos value",
// 			envName:       "NODE_OPTIONS",
// 			observedValue: nodeOptionsNativeCommunity,
// 			wantRevert:    false,
// 		},
// 		{
// 			name:          "user value with odigos value matching SDKs",
// 			envName:       "NODE_OPTIONS",
// 			observedValue: userVal + " " + nodeOptionsNativeCommunity,
// 			wantRevert:    true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gotRevert := ShouldRevert(tt.envName, tt.observedValue)
// 			assert.Equal(t, tt.wantRevert, gotRevert, "mismatch in ShouldRevert: %s", tt.name)
// 		})
// 	}
// }
