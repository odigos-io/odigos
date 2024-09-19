package versionsupport

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestIsRuntimeVersionSupported_NotSupported(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
		errorMsg string
	}{
		//{"java version not supported", common.JavaProgrammingLanguage, "17.0.10+9", "java runtime version not supported by OpenTelemetry SDK. Found: 17.0.10+9, supports: 17.0.11+8"},
		//{"jdk version not supported", common.JavaProgrammingLanguage, "17.0.11+7", "java runtime version not supported by OpenTelemetry SDK. Found: 17.0.11+7, supports: 17.0.11+8"},
		{"go version not supported", common.GoProgrammingLanguage, "1.14", "go runtime version not supported by OpenTelemetry SDK. Found: 1.14, supports: 1.17.0"},
		{"javascript version not supported", common.JavascriptProgrammingLanguage, "13.9.9", "javascript runtime version not supported by OpenTelemetry SDK. Found: 13.9.9, supports: 14.0.0"},
		{"python version not supported", common.PythonProgrammingLanguage, "3.7", "python runtime version not supported by OpenTelemetry SDK. Found: 3.7, supports: 3.8.0"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.25.4", "nginx runtime version not supported by OpenTelemetry SDK. Found: 1.25.4, supports: 1.25.5, 1.26.0"},
		{"nginx version not supported", common.NginxProgrammingLanguage, "1.26.1", "nginx runtime version not supported by OpenTelemetry SDK. Found: 1.26.1, supports: 1.25.5, 1.26.0"},
		// {"unknown language", common.UnknownProgrammingLanguage, "0.0.0", "Unsupported language: unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			runtimeDetails := []v1alpha1.RuntimeDetailsByContainer{
				{Language: tc.language, RuntimeVersion: tc.version},
			}

			supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
			assert.Equal(t, false, supported)
			assert.Error(t, err)
			assert.Equal(t, err.Error(), tc.errorMsg)
		})
	}
}

func TestIsRuntimeVersionSupported_Support(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		//{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
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

			supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
			assert.Equal(t, true, supported)
			assert.Nil(t, err)
		})
	}
}

func TestIsRuntimeVersionSupported_MultiRuntimeContainer_NotSupport(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		//{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
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
		supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
		assert.Equal(t, false, supported)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "python runtime version not supported by OpenTelemetry SDK. Found: 3.7, supports: 3.8.0")
	})
}

func TestIsRuntimeVersionSupported_MultiRuntimeContainer_Support(t *testing.T) {
	testCases := []struct {
		name     string
		language common.ProgrammingLanguage
		version  string
	}{
		//{"java version not supported", common.JavaProgrammingLanguage, "17.0.11+9"},
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
		supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
		assert.Equal(t, true, supported)
		assert.Nil(t, err)
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

		supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
		assert.Equal(t, false, supported)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "Unsupported language: ruby")
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

		supported, err := IsRuntimeVersionSupported(nil, runtimeDetails)
		assert.Equal(t, true, supported)
		assert.Nil(t, err)
	})
}
