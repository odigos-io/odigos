package java

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

// libjvmRegex is a regular expression that matches any path containing "libjvm.so",
// ensuring that we correctly detect the presence of the JVM shared library.
var libjvmRegex = regexp.MustCompile(`.*/libjvm\.so`)

const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`
const processName = "java"

var re = regexp.MustCompile(JavaVersionRegex)

// Note: We could support GraalVM native images in the future if needed.
// GraalVM native images do not load libjvm.so but have Graal-specific arguments.
// This could be detected with something like: strings.Contains(proc.CmdLine, "-XX:+UseGraalVM") || strings.Contains(proc.CmdLine, "-H:+")
func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if checkForLoadedJVM(proc.ProcessID) {
		return common.JavaProgrammingLanguage, true
	}

	// TODO: do we need to cover the case that the process is a Java process but it dont loaded JVM?
	if filepath.Base(proc.ExePath) == processName {
		return common.JavaProgrammingLanguage, true
	}

	// TODO: (optional) add support for GraalVM
	// if isGraalVMProcess(proc.CmdLine) {
	// 	return common.JavaProgrammingLanguage, true
	// }

	return "", false
}

// This function inspects the memory-mapped regions of the process by reading the "/proc/<pid>/maps" file.
// It then searches for "libjvm.so", which is a shared library loaded by Java processes.
func checkForLoadedJVM(processID int) bool {
	mapsPath := fmt.Sprintf("/proc/%d/maps", processID)
	mapsFile, err := os.Open(mapsPath)
	if err != nil {
		return false
	}
	defer mapsFile.Close()

	// Look for shared JVM libraries line by line inside the "/proc/<pid>/maps" file
	scanner := bufio.NewScanner(mapsFile)
	for scanner.Scan() {
		if libjvmRegex.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
