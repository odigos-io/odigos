//go:build linux

package runtime_details

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestMultipleLanguagesDetected_SeparateContainers_NoConflict(t *testing.T) {
	t.Parallel()

	// Arrange: two containers, each with a single (different) language — no conflict
	results := InspectionResults{}

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages(containerALangs, &results)
	collectDetectedLanguages(containerBLangs, &results)

	// Assert: no multi-language conflict — each container has exactly one language
	assert.False(t, results.multipleLanguagesDetected)
	assert.Len(t, results.detectedLanguages, 2)
}

func TestMultipleLanguagesDetected_TwoContainersWithConflicts(t *testing.T) {
	// Arrange: two containers, each with a multi-language conflict
	results := InspectionResults{}

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
		101: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.JavascriptProgrammingLanguage, RuntimeVersion: "18.0.0"},
		201: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages(containerALangs, &results)
	collectDetectedLanguages(containerBLangs, &results)

	// Assert: both containers have multi-language conflicts
	assert.True(t, results.multipleLanguagesDetected)
	assert.Len(t, results.detectedLanguages, 4)
}
