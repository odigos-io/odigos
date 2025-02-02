package java

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"
const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var re = regexp.MustCompile(JavaVersionRegex)

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	exe: filepath.Base(proc.ExePath)
	if (proc.ExePath)
	if strings.Contains(proc.ExePath, processName) || strings.Contains(proc.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func isJavaProcess(proc *ProcessInfo) bool {
    // Check if executable is java binary
    if isJavaExecutable(proc.ExePath) {
        return true
    }

    // Parse command line safely (split by null bytes)
    args := strings.Split(proc.CmdLine, "\x00")
    if len(args) == 0 {
        return false
    }

    // Check if first argument is java
    firstArg := filepath.Base(args[0])
    if firstArg == "java" || firstArg == "javaw" {
        return true
    }

    // Look for specific Java indicators
    for _, arg := range args {
        // Check for JVM specific flags
        if strings.HasPrefix(arg, "-Xmx") ||
           strings.HasPrefix(arg, "-Xms") ||
           strings.HasPrefix(arg, "-Djava.") ||
           strings.HasPrefix(arg, "-XX:") {
            return true
        }
    }

    return false
}

func isJavaExecutable(path string) bool {
    exe := filepath.Base(path)
	
    return exe == "java" || exe == "javaw"
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
