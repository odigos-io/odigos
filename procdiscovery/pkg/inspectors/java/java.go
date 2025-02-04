package java

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

// libjvmRegex is a regular expression that matches any path containing "libjvm.so",
// ensuring that we correctly detect the presence of the JVM shared library.
var libjvmRegex = regexp.MustCompile(`.*/libjvm\.so`)

const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var re = regexp.MustCompile(JavaVersionRegex)

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if checkForLoadedJVM(proc.ProcessID) {
		return common.JavaProgrammingLanguage, true
	}

	// 3. Check if the executable file is "java" or "javaw"
	if isJavaExecutable(proc.ExePath) {
		return common.JavaProgrammingLanguage, true
	}

	if isGraalVMProcess(proc.CmdLine) {
		return common.JavaProgrammingLanguage, true
	}
	return "", false
}

// This function inspects the memory-mapped regions of the process by reading the "/proc/<pid>/maps" file.
// It then searches for "libjvm.so", which is a shared library loaded by Java processes.
func checkForLoadedJVM(pid int) bool {
	mapsPath := fmt.Sprintf("/proc/%d/maps", pid)
	mapsBytes, err := os.ReadFile(mapsPath)
	if err != nil {
		return false
	}

	// Look for shared JVM libraries
	mapsStr := string(mapsBytes)
	return libjvmRegex.MatchString(mapsStr)
}

// isJavaExecutable checks if the process binary name suggests it's a Java process.
// This is useful for cases where "libjvm.so" isn't found in "/proc/<pid>/maps".
func isJavaExecutable(procExe string) bool {
	return strings.HasSuffix(procExe, "java")
}

func isGraalVMProcess(cmdline string) bool {
	// GraalVM native images do not load libjvm.so but have Graal-specific arguments
	return strings.Contains(cmdline, "-XX:+UseGraalVM") || strings.Contains(cmdline, "-H:+")
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
