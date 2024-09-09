package versionsupport

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsRuntimeVersionSupported_NotSupported(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		{"java version not supported", common.JavaProgrammingLanguage, "17.0.10+9"},
		{"jdk version not supported", common.JavaProgrammingLanguage, "17.0.11+7"},
		{"go version not supported", common.GoProgrammingLanguage, "1.14"},
		{"javascript version not supported", common.JavascriptProgrammingLanguage, "13.9.9"},
		{"python version not supported", common.PythonProgrammingLanguage, "3.7"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.25.4"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.26.1"},
		{"unknown language", common.UnknownProgrammingLanguage, "0.0.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{
				{Language: tc.language, RuntimeVersion: tc.version},
			}

			supported := IsRuntimeVersionSupported(runtimeDetails)
			assert.Equal(t, false, supported)
		})
	}
}

func TestIsRuntimeVersionSupported_Support(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
		{"go version not supported", common.GoProgrammingLanguage, "1.18"},
		{"dotnet version not supported", common.DotNetProgrammingLanguage, "0.0.0"},
		{"javascript version not supported", common.JavascriptProgrammingLanguage, "14.0.1"},
		{"python version not supported", common.PythonProgrammingLanguage, "3.9"},
		{"mySQL version not supported", common.MySQLProgrammingLanguage, "1.14.1"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.25.5"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.26"},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {

			runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{
				{Language: tc.language, RuntimeVersion: tc.version},
			}

			supported := IsRuntimeVersionSupported(runtimeDetails)
			assert.Equal(t, true, supported)
		})
	}
}

func TestIsRuntimeVersionSupported_MultiRuntimeContainer_NotSupport(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
		{"go version not supported", common.GoProgrammingLanguage, "1.18"},
		{"dotnet version not supported", common.DotNetProgrammingLanguage, "0.0.0"},
		{"javascript version not supported", common.JavascriptProgrammingLanguage, "14.0.1"},
		{"python version not supported", common.PythonProgrammingLanguage, "3.7"},
		{"mySQL version not supported", common.MySQLProgrammingLanguage, "1.14.1"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.25.5"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.26"},
	}

	runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{}

	for _, tc := range testCases {
		runtimeDetails = append(runtimeDetails, v1alpha1.RuntimeDetailsByContainer{
			Language:       tc.language,
			RuntimeVersion: tc.version,
		})
	}

	// Run the test for the combined runtimeDetails
	t.Run("multiple runtimes not supported", func(t *testing.T) {
		supported := IsRuntimeVersionSupported(runtimeDetails)
		assert.Equal(t, false, supported)
	})
}

func TestIsRuntimeVersionSupported_MultiRuntimeContainer_Support(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
		{"go version not supported", common.GoProgrammingLanguage, "1.18"},
		{"dotnet version not supported", common.DotNetProgrammingLanguage, "0.0.0"},
		{"javascript version not supported", common.JavascriptProgrammingLanguage, "14.0.1"},
		{"python version not supported", common.PythonProgrammingLanguage, "3.9"},
		{"mySQL version not supported", common.MySQLProgrammingLanguage, "1.14.1"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.25.5"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.26"},
	}

	runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{}

	for _, tc := range testCases {
		runtimeDetails = append(runtimeDetails, v1alpha1.RuntimeDetailsByContainer{
			Language:       tc.language,
			RuntimeVersion: tc.version,
		})
	}

	// Run the test for the combined runtimeDetails
	t.Run("multiple runtimes not supported", func(t *testing.T) {
		supported := IsRuntimeVersionSupported(runtimeDetails)
		assert.Equal(t, true, supported)
	})
}

func TestIsRuntimeVersionSupported_LanguageDoesNotExist_NotSupported(t *testing.T) {
	testCase := struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{"ruby language is not supported", "ruby", "17.0.10+9"}

	t.Run(testCase.name, func(t *testing.T) {

		runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{
			{Language: testCase.language, RuntimeVersion: testCase.version},
		}

		supported := IsRuntimeVersionSupported(runtimeDetails)
		assert.Equal(t, false, supported)
	})
}

func TestIsRuntimeVersionSupported_VersionNotFound_Supported(t *testing.T) {
	testCase := struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{"java version not supported", common.JavaProgrammingLanguage, ""}

	t.Run(testCase.name, func(t *testing.T) {

		runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{
			{Language: testCase.language, RuntimeVersion: testCase.version},
		}

		supported := IsRuntimeVersionSupported(runtimeDetails)
		assert.Equal(t, true, supported)
	})
}
