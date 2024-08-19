package common

type ProgramLanguageDetails struct {
	Language       ProgrammingLanguage
	RuntimeVersion string
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
