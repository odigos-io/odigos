package common

// +kubebuilder:validation:Enum=java;python;go;dotnet;javascript;mysql
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

	// This is a special value that is used when the language is not detected
	UnknownProgrammingLanguage ProgrammingLanguage = "unknown"
)
