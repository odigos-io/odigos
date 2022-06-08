package inspectors

import (
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
	"strings"
)

type javaInspector struct{}

var java = &javaInspector{}

const processName = "java"

func (j *javaInspector) Inspect(p *process.Details) (v1.ProgrammingLanguage, bool) {
	if strings.Contains(p.ExeName, processName) || strings.Contains(p.CmdLine, processName) {
		return v1.JavaProgrammingLanguage, true
	}

	return "", false
}
