package inspectors

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/langDetector/process"
	"os"
	"strings"
)

type javaInspector struct{}

var java = &javaInspector{}

const processName = "java"
const hsperfdataDir = "hsperfdata"

func (j *javaInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(p.ExeName, processName) || strings.Contains(p.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	if j.searchForHsperfdata(p.ProcessID) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *javaInspector) searchForHsperfdata(pid int) bool {
	tmpDir := fmt.Sprintf("/proc/%d/root/tmp/", pid)
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return false
	}

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			if strings.Contains(name, hsperfdataDir) {
				return true
			}
		}
	}
	return false
}
