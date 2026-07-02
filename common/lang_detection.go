package common

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ProgramLanguageDetails struct {
	Language       ProgrammingLanguage
	RuntimeVersion string
}

// +kubebuilder:validation:Enum=java;python;go;dotnet;javascript;browser;php;ruby;rust;cplusplus;mysql;nginx;redis;postgres;unknown;ignored;*
type ProgrammingLanguage string

const (
	// ProgrammingLanguageWildcard means the distribution accepts any container programming language (see distro YAML).
	ProgrammingLanguageWildcard   ProgrammingLanguage = "*"
	JavaProgrammingLanguage       ProgrammingLanguage = "java"
	PythonProgrammingLanguage     ProgrammingLanguage = "python"
	GoProgrammingLanguage         ProgrammingLanguage = "go"
	DotNetProgrammingLanguage     ProgrammingLanguage = "dotnet"
	JavascriptProgrammingLanguage ProgrammingLanguage = "javascript"
	// BrowserProgrammingLanguage is JavaScript that runs in the end-user's browser (the OpenTelemetry
	// Web SDK), as opposed to JavascriptProgrammingLanguage which is server-side Node.js. Unlike the
	// other languages, the code is not executed by a process inside the pod, so it cannot be detected
	// via /proc and is delivered to the browser by the odigos-browser-proxy sidecar.
	BrowserProgrammingLanguage   ProgrammingLanguage = "browser"
	PhpProgrammingLanguage       ProgrammingLanguage = "php"
	RubyProgrammingLanguage      ProgrammingLanguage = "ruby"
	RustProgrammingLanguage      ProgrammingLanguage = "rust"
	CPlusPlusProgrammingLanguage ProgrammingLanguage = "cplusplus"
	CSharpProgrammingLanguage    ProgrammingLanguage = "csharp"
	SwiftProgrammingLanguage     ProgrammingLanguage = "swift"
	ElixirProgrammingLanguage    ProgrammingLanguage = "elixir"
	// This is an experimental feature, It is not a language
	// but in order to avoid huge refactoring we are adding it here for now
	MySQLProgrammingLanguage    ProgrammingLanguage = "mysql"
	NginxProgrammingLanguage    ProgrammingLanguage = "nginx"
	RedisProgrammingLanguage    ProgrammingLanguage = "redis"
	PostgresProgrammingLanguage ProgrammingLanguage = "postgres"
	// Used when the language detection is not successful for all the available inspectors
	UnknownProgrammingLanguage ProgrammingLanguage = "unknown"
)

// IsProgrammingLanguageWildcard reports whether lang is the distro wildcard meaning "any language".
func IsProgrammingLanguageWildcard(lang ProgrammingLanguage) bool {
	return strings.TrimSpace(string(lang)) == string(ProgrammingLanguageWildcard)
}

// MapOdigosToSemConv maps odigos programming language to OpenTelemetry semantic conventions
// It is supported only for the languages that are supported by OpenTelemetry [not for mysql, nginx, etc.]
func MapOdigosToSemConv(odigosPrograminglang ProgrammingLanguage) string {
	switch odigosPrograminglang {
	case JavascriptProgrammingLanguage:
		return semconv.TelemetrySDKLanguageNodejs.Value.AsString()
	case BrowserProgrammingLanguage:
		return semconv.TelemetrySDKLanguageWebjs.Value.AsString()
	default:
		return string(odigosPrograminglang)
	}
}

func GetVersion(versionString string) *version.Version {
	return ParseRuntimeVersion(versionString)
}

// ParseRuntimeVersion parses a runtime version for semver constraint checks.
// Leading "v" and prerelease/build suffixes (e.g. "v1.2.3-0", "1.2.3+build") are normalized
// to the core release (1.2.3) so they compare correctly against supported version ranges.
func ParseRuntimeVersion(versionString string) *version.Version {
	v, err := version.NewVersion(versionString)
	if err != nil {
		return nil
	}
	return v.Core()
}

func MajorMinorStringOnly(v *version.Version) (string, error) {
	segments := v.Segments()
	if len(segments) < 2 {
		// fallback for malformed versions
		return "", fmt.Errorf("version %s has less than 2 segments", v.String())
	}
	return fmt.Sprintf("%d.%d", segments[0], segments[1]), nil
}
