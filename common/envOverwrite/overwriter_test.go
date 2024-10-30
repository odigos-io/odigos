package envOverwrite

import (
	"fmt"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestGetPatchedEnvValue(t *testing.T) {
	nodeOptionsNativeCommunity, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkNativeCommunity)
	nodeOptionsEbpfEnterprise, _ := ValToAppend("NODE_OPTIONS", common.OtelSdkEbpfEnterprise)
	javaToolsNativeCommunity, _ := ValToAppend("JAVA_TOOL_OPTIONS", common.OtelSdkNativeCommunity)
	userVal := "--max-old-space-size=4096"
	specialEnvValueJava := "-javaagent:/opt/sre-agent/sre-agent.jar"

	// test different cases
	tests := []struct {
		name                 string
		envName              string
		observedValue        string
		sdk                  *common.OtelSdk
		programmingLanguage  common.ProgrammingLanguage
		patchedValueExpected string
	}{
		{
			name:                 "un-relevant env var",
			envName:              "PATH",
			observedValue:        "/usr/local/bin:/usr/bin:/bin",
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: "",
		},
		{
			name:                 "only user value",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal,
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: userVal + " " + nodeOptionsNativeCommunity,
		},
		{
			name:                 "only odigos value",
			envName:              "NODE_OPTIONS",
			observedValue:        nodeOptionsNativeCommunity,
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: "",
		},
		{
			name:                 "user value with odigos value matching SDKs",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal + " " + nodeOptionsNativeCommunity,
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: userVal + " " + nodeOptionsNativeCommunity,
		},
		{
			name:                 "user value with odigos value with different SDKs",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal + " " + nodeOptionsNativeCommunity,
			sdk:                  &common.OtelSdkEbpfEnterprise,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: userVal + " " + nodeOptionsEbpfEnterprise,
		},
		{
			// No user values are observed, hence there is not need to patch
			// even if the observed value is different from the SDK value
			name:                 "observed odigos value different from SDK",
			envName:              "NODE_OPTIONS",
			observedValue:        nodeOptionsNativeCommunity,
			sdk:                  &common.OtelSdkEbpfEnterprise,
			programmingLanguage:  common.JavascriptProgrammingLanguage,
			patchedValueExpected: "",
		},
		{
			name:                 "observed env is for a different programming language than what detected",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal,
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.PythonProgrammingLanguage,
			patchedValueExpected: "",
		},
		{
			name:                 "no otel sdk (unknown language or ignored container)",
			envName:              "NODE_OPTIONS",
			observedValue:        userVal,
			sdk:                  nil,
			programmingLanguage:  common.UnknownProgrammingLanguage,
			patchedValueExpected: "",
		},
		{
			name:                 "multiple values in env var",
			envName:              "JAVA_TOOL_OPTIONS",
			observedValue:        fmt.Sprintf("%s %s %s %s", specialEnvValueJava, specialEnvValueJava, specialEnvValueJava, javaToolsNativeCommunity),
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavaProgrammingLanguage,
			patchedValueExpected: specialEnvValueJava + " " + javaToolsNativeCommunity,
		},
		{
			name:                 "multiple spaces in special env value",
			envName:              "JAVA_TOOL_OPTIONS",
			observedValue:        fmt.Sprintf("%s %s              %s", specialEnvValueJava, specialEnvValueJava, javaToolsNativeCommunity),
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavaProgrammingLanguage,
			patchedValueExpected: specialEnvValueJava + " " + javaToolsNativeCommunity,
		},
		{
			name:                 "tabs in special env value",
			envName:              "JAVA_TOOL_OPTIONS",
			observedValue:        fmt.Sprintf("%s \t %s \t %s", specialEnvValueJava, specialEnvValueJava, javaToolsNativeCommunity),
			sdk:                  &common.OtelSdkNativeCommunity,
			programmingLanguage:  common.JavaProgrammingLanguage,
			patchedValueExpected: specialEnvValueJava + " " + javaToolsNativeCommunity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchedValue := GetPatchedEnvValue(tt.envName, tt.observedValue, tt.sdk, tt.programmingLanguage)
			if patchedValue == nil {
				assert.Equal(t, tt.patchedValueExpected, "", "mismatch in GetPatchedEnvValue: %s", tt.name)
			} else {
				assert.Equal(t, tt.patchedValueExpected, *patchedValue, "mismatch in GetPatchedEnvValue: %s", tt.name)
			}
		})
	}

}
