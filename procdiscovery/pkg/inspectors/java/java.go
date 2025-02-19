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

func (j *JavaInspector) LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// Low cost: simply check the exe filename.
	if filepath.Base(ctx.ExePath) == processName {
		return common.JavaProgrammingLanguage, true
	}
	return "", false
}

func (j *JavaInspector) ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	ctx.MapsContent()
	if ctx.MapsFileContent == nil {
		return "", false
	}
	scanner := bufio.NewScanner(ctx.MapsFileContent)
	for scanner.Scan() {
		if libjvmRegex.MatchString(scanner.Text()) {
			return common.JavaProgrammingLanguage, true
		}
	}
	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(ctx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := ctx.Details.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
