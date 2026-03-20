package datacollectorcfg

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
