package common

// +kubebuilder:validation:Enum=java;python;go;dotnet;javascript;mysql;unknown
type ProgrammingLanguage string

const (
	JavaProgrammingLanguage       ProgrammingLanguage = "java"
	PythonProgrammingLanguage     ProgrammingLanguage = "python"
	GoProgrammingLanguage         ProgrammingLanguage = "go"
	DotNetProgrammingLanguage     ProgrammingLanguage = "dotnet"
	JavascriptProgrammingLanguage ProgrammingLanguage = "javascript"
	// This is an experimental feature, It is not a language
	// but in order to avoid huge refactoring we are adding it here for now
	MySQLProgrammingLanguage      ProgrammingLanguage = "mysql"
	// Used when the language detection is not successful for all the available inspectors
	UnknownProgrammingLanguage    ProgrammingLanguage = "unknown"
)
