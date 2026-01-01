package java

import (
	"regexp"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

var (
	processNames = []string{
		"java",
	}
	binaries = []string{
		"/libjvm.so", // Ensures "libjvm.so" appears within a path (because of the "/" prefix)
	}
	// versionRegex matches JAVA_VERSION format like "11.0.2+9" or "17.0.1.1+12"
	versionRegex = regexp.MustCompile(`\d+\.\d+\.\d+(?:\.\d+)?\+\d+`)
	// javaHomeVersionRegex extracts version from JAVA_HOME paths like "/usr/lib/jvm/zulu7.56.0.11-ca-jdk7.0.352-linux_x64"
	// It looks for "jdk" followed by a version number
	javaHomeVersionRegex = regexp.MustCompile(`jdk(\d+(?:\.\d+)*)`)
)

func (j *JavaInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if utils.IsProcessEqualProcessNames(pcx, processNames) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}

	if utils.IsMapsFileContainsBinary(mapsFile, binaries) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	javaVersion := ""
	if value, exists := pcx.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion = versionRegex.FindString(value)
	}

	// Prefer JAVA_VERSION to have backward compatibility
	javaHome := j.GetJavaHome(pcx)
	if javaHome != "" {
		println("javaHome", javaHome)
		subMatch := javaHomeVersionRegex.FindStringSubmatch(javaHome)
		println("subMatch", subMatch)
		if len(subMatch) > 1 {
			javaVersion = subMatch[1]
			println("javaVersion", javaVersion)
		}
	}

	return javaVersion
}

func (j *JavaInspector) GetJavaHome(pcx *process.ProcessContext) string {
	println("Getting Java Home")
	if value, exists := pcx.GetDetailedEnvsValue(process.JavaHomeConst); exists {
		println("Java Home", value)
		return value
	}

	println("No Java Home found")
	return ""
}
