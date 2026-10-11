package envOverwrite

import (
	"github.com/odigos-io/odigos/common"
)

// EnvVarsForLanguage is a map of environment variables that are relevant for each language.
var EnvVarsForLanguage = map[common.ProgrammingLanguage][]string{
	common.JavascriptProgrammingLanguage: {"NODE_OPTIONS"},
	common.PythonProgrammingLanguage:     {"PYTHONPATH"},
	common.JavaProgrammingLanguage:       {"JAVA_TOOL_OPTIONS"},
}
