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

// SpanKind is already defined in opentelemetry-go as int.
// this value can go into the CRD in which case it will be string for user convenience.
// +kubebuilder:validation:Enum=client;server;producer;consumer;internal
type SpanKind string

const (
	ClientSpanKind   SpanKind = "client"
	ServerSpanKind   SpanKind = "server"
	ProducerSpanKind SpanKind = "producer"
	ConsumerSpanKind SpanKind = "consumer"
	InternalSpanKind SpanKind = "internal"
)
