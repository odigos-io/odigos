package actions

// These are the valid conditions for an odigos action
const (
	// TransformedToProcessor is the condition when the action CR is transformed to a processor CR.
	// This is the first step in the reconciliation process.
	ActionTransformedToProcessorType = "TransformedToProcessor"
)

// Reasons for action condition types
const (
	//
	// ActionTransformedToProcessor:

	// ProcessorCreatedReason is added to the action when the processor CR is created.
	ProcessorCreatedReason = "ProcessorCreated"
	// FailedToCreateProcessorReason is added to the action when the processor CR creation fails.
	FailedToCreateProcessorReason = "FailedToCreateProcessor"
	// FailedToTransformToProcessorReason is added to the action when the transformation to processor object fails.
	FailedToTransformToProcessorReason = "FailedToTransformToProcessor"
)

type OttlStatementConfig struct {
	Context    string   `json:"context"`
	Statements []string `json:"statements"`
}

type TransformProcessorConfig struct {
	ErrorMode        string                `json:"error_mode"`
	TraceStatements  []OttlStatementConfig `json:"trace_statements,omitempty"`
	MetricStatements []OttlStatementConfig `json:"metric_statements,omitempty"`
	LogStatements    []OttlStatementConfig `json:"log_statements,omitempty"`
}
