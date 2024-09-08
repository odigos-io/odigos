package common

import (
	"github.com/hashicorp/go-version"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ProgramLanguageDetails struct {
	Language       ProgrammingLanguage
	RuntimeVersion *version.Version
}

// +kubebuilder:validation:Enum=java;python;go;dotnet;javascript;mysql;nginx;unknown;ignored
type ProgrammingLanguage string

const (
	JavaProgrammingLanguage       ProgrammingLanguage = "java"
	PythonProgrammingLanguage     ProgrammingLanguage = "python"
	GoProgrammingLanguage         ProgrammingLanguage = "go"
	DotNetProgrammingLanguage     ProgrammingLanguage = "dotnet"
	JavascriptProgrammingLanguage ProgrammingLanguage = "javascript"
	// This is an experimental feature, It is not a language
	// but in order to avoid huge refactoring we are adding it here for now
	MySQLProgrammingLanguage ProgrammingLanguage = "mysql"
	NginxProgrammingLanguage ProgrammingLanguage = "nginx"
	// Used when the language detection is not successful for all the available inspectors
	UnknownProgrammingLanguage ProgrammingLanguage = "unknown"
	// Ignored is used when the odigos is configured to ignore the process/container
	IgnoredProgrammingLanguage ProgrammingLanguage = "ignored"
)

// MapOdigosToSemConv maps odigos programming language to OpenTelemetry semantic conventions
// It is supported only for the languages that are supported by OpenTelemetry [not for mysql, nginx, etc.]
func MapOdigosToSemConv(odigosPrograminglang ProgrammingLanguage) string {
	switch odigosPrograminglang {
	case JavascriptProgrammingLanguage:
		return semconv.TelemetrySDKLanguageNodejs.Value.AsString()
	default:
		return string(odigosPrograminglang)
	}
}

func GetVersion(versionString string) *version.Version {
	runtimeVersion, err := version.NewVersion(versionString)
	if err != nil {
		return nil
	}
	return runtimeVersion
}
