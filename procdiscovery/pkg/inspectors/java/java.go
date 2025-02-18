package java

import (
	"bufio"
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

const processName = "java"
const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var re = regexp.MustCompile(JavaVersionRegex)

func (j *JavaInspector) InspectLow(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// Low cost: simply check the exe filename.
	if filepath.Base(ctx.ExePath) == processName {
		return common.JavaProgrammingLanguage, true
	}
	return "", false
}

func (j *JavaInspector) InspectHeavy(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// Heavy cost: use cached maps content.
	maps := ctx.MapsContent()
	if maps == nil {
		return "", false
	}
	scanner := bufio.NewScanner(maps)
	for scanner.Scan() {
		if libjvmRegex.MatchString(scanner.Text()) {
			return common.JavaProgrammingLanguage, true
		}
	}
	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}

/////////////////////////////////////////////////////////////////////////

// // Note: We could support GraalVM native images in the future if needed.
// // GraalVM native images do not load libjvm.so but have Graal-specific arguments.
// // This could be detected with something like: strings.Contains(proc.CmdLine, "-XX:+UseGraalVM") || strings.Contains(proc.CmdLine, "-H:+")
// func (j *JavaInspector) Inspect(processContext *process.ProcessContext) (common.ProgrammingLanguage, bool) {

// 	mapsFile := processContext.MapsContent()
// 	if mapsFile == nil {
// 		return "", false
// 	}

// 	scanner := bufio.NewScanner(mapsFile)
// 	for scanner.Scan() {
// 		if libjvmRegex.MatchString(scanner.Text()) {
// 			return common.JavaProgrammingLanguage, true
// 		}
// 	}

// 	// TODO: do we need to cover the case that the process is a Java process but it dont loaded JVM?
// 	if filepath.Base(processContext.ExePath) == processName {
// 		return common.JavaProgrammingLanguage, true
// 	}

// 	return "", false
// }

// // This function inspects the memory-mapped regions of the process by reading the "/proc/<pid>/maps" file.
// // It then searches for "libjvm.so", which is a shared library loaded by Java processes.
// func checkForLoadedJVM(processID int) bool {
// 	mapsPath := fmt.Sprintf("/proc/%d/maps", processID)
// 	mapsFile, err := os.Open(mapsPath)
// 	if err != nil {
// 		return false
// 	}
// 	defer mapsFile.Close()

// 	// Look for shared JVM libraries line by line inside the "/proc/<pid>/maps" file
// 	scanner := bufio.NewScanner(mapsFile)
// 	for scanner.Scan() {
// 		if libjvmRegex.MatchString(scanner.Text()) {
// 			return true
// 		}
// 	}
// 	return false
// }
