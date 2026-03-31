package runtime_details

import (
	"errors"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

// ******************************************
// SelectContainerMainLanguage tests
// ******************************************

func Test_SelectMainLanguage_NoLanguages_ReturnsError(t *testing.T) {
	// no known languages detected — should fail with the sentinel error (before the heuristics come into play)
	languages := []common.ProgrammingLanguage{}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.ErrorIs(t, selectionError, errNoKnownLanguageDetected)
	assert.Equal(t, common.UnknownProgrammingLanguage, result)
}

func Test_SelectMainLanguage_SingleLanguage_ReturnsThatLanguage(t *testing.T) {
	// exactly one language detected — should return it directly
	languages := []common.ProgrammingLanguage{common.GoProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.GoProgrammingLanguage, result)
}

func Test_SelectMainLanguage_JavaWithHighPriority_JavaSelected(t *testing.T) {
	// Java paired with another high-priority language — Java always wins
	languages := []common.ProgrammingLanguage{common.JavaProgrammingLanguage, common.GoProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.JavaProgrammingLanguage, result)
}

func Test_SelectMainLanguage_HighPriorityWithJava_JavaSelected(t *testing.T) {
	// Java in second position — should still be selected over the other language
	languages := []common.ProgrammingLanguage{common.DotNetProgrammingLanguage, common.JavaProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.JavaProgrammingLanguage, result)
}

func Test_SelectMainLanguage_LowPriorityCppFirst_HighPrioritySelected(t *testing.T) {
	// C++ (low priority) paired with Go — Go should be selected
	languages := []common.ProgrammingLanguage{common.CPlusPlusProgrammingLanguage, common.GoProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.GoProgrammingLanguage, result)
}

func Test_SelectMainLanguage_LowPriorityNginxSecond_HighPrioritySelected(t *testing.T) {
	// Nginx (low priority) in second position paired with DotNet — DotNet wins
	languages := []common.ProgrammingLanguage{common.DotNetProgrammingLanguage, common.NginxProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.DotNetProgrammingLanguage, result)
}

func Test_SelectMainLanguage_LowPriorityPythonFirst_HighPrioritySelected(t *testing.T) {
	// Python (low priority) paired with JavaScript — JavaScript wins
	languages := []common.ProgrammingLanguage{common.PythonProgrammingLanguage, common.JavascriptProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.JavascriptProgrammingLanguage, result)
}

func Test_SelectMainLanguage_TwoLowPriorityLanguages_SecondSelected(t *testing.T) {
	// both languages are low priority — the first is deprioritized so the second wins
	languages := []common.ProgrammingLanguage{common.CPlusPlusProgrammingLanguage, common.PythonProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.PythonProgrammingLanguage, result)
}

func Test_SelectMainLanguage_TwoHighPriorityNonJava_ReturnsError(t *testing.T) {
	// two high-priority non-Java languages — ambiguous, should error
	languages := []common.ProgrammingLanguage{common.GoProgrammingLanguage, common.DotNetProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.Error(t, selectionError)
	assert.False(t, errors.Is(selectionError, errNoKnownLanguageDetected))
	assert.Equal(t, common.UnknownProgrammingLanguage, result)
}

func Test_SelectMainLanguage_ThreeLanguages_ReturnsError(t *testing.T) {
	// more than two languages — always ambiguous regardless of which languages
	languages := []common.ProgrammingLanguage{common.GoProgrammingLanguage, common.JavaProgrammingLanguage, common.PythonProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.Error(t, selectionError)
	assert.False(t, errors.Is(selectionError, errNoKnownLanguageDetected))
	assert.Equal(t, common.UnknownProgrammingLanguage, result)
}

func Test_SelectMainLanguage_JavaWithLowPriority_JavaSelected(t *testing.T) {
	// Java paired with a low-priority language — Java rule takes precedence over low-priority rule
	languages := []common.ProgrammingLanguage{common.NginxProgrammingLanguage, common.JavaProgrammingLanguage}

	result, selectionError := SelectContainerMainLanguage(languages)

	assert.NoError(t, selectionError)
	assert.Equal(t, common.JavaProgrammingLanguage, result)
}
