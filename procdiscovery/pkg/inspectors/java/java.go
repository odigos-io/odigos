package java

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"

func (j *JavaInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(p.ExeName, processName) || strings.Contains(p.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}
