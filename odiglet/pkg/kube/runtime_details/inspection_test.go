//go:build linux

package runtime_details

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func newInspectionResults() InspectionResults {
	return InspectionResults{
		containerDetectedLanguages: make(map[string][]common.ProgramLanguageDetails),
	}
}

func TestCollectDetectedLanguages_SeparateContainers_NoConflict(t *testing.T) {
	t.Parallel()

	// Arrange: two containers, each with a single (different) language — no conflict
	results := newInspectionResults()

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages("container-a", containerALangs, &results)
	collectDetectedLanguages("container-b", containerBLangs, &results)

	// Assert: no multi-language conflict — each container has exactly one language
	assert.Len(t, results.containerDetectedLanguages, 2)
	assert.Len(t, results.containerDetectedLanguages["container-a"], 1)
	assert.Equal(t, common.JavaProgrammingLanguage, results.containerDetectedLanguages["container-a"][0].Language)
	assert.Len(t, results.containerDetectedLanguages["container-b"], 1)
	assert.Equal(t, common.PythonProgrammingLanguage, results.containerDetectedLanguages["container-b"][0].Language)
}

func TestCollectDetectedLanguages_TwoContainersWithConflicts(t *testing.T) {
	t.Parallel()

	// Arrange: two containers, each with a multi-language conflict
	results := newInspectionResults()

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
		101: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.JavascriptProgrammingLanguage, RuntimeVersion: "18.0.0"},
		201: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages("container-a", containerALangs, &results)
	collectDetectedLanguages("container-b", containerBLangs, &results)

	// Assert: both containers have multi-language conflicts
	assert.Len(t, results.containerDetectedLanguages, 2)
	assert.Len(t, results.containerDetectedLanguages["container-a"], 2)
	assert.Len(t, results.containerDetectedLanguages["container-b"], 2)
}
