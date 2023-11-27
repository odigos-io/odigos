package common

type LanguageByContainer struct {
	ContainerName string              `json:"containerName"`
	Language      ProgrammingLanguage `json:"language"`
	ProcessName   string              `json:"processName,omitempty"`
}

// +kubebuilder:validation:Enum=java;python;go;dotnet;javascript
type ProgrammingLanguage string

const (
	JavaProgrammingLanguage       ProgrammingLanguage = "java"
	PythonProgrammingLanguage     ProgrammingLanguage = "python"
	GoProgrammingLanguage         ProgrammingLanguage = "go"
	DotNetProgrammingLanguage     ProgrammingLanguage = "dotnet"
	JavascriptProgrammingLanguage ProgrammingLanguage = "javascript"
)
