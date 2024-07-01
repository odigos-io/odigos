package sampling

const (
	DefaultWaitDuraiton = "30s"
)

type GroupByTraceConfig struct {
	WaitDuration string `json:"wait_duration"`
}
