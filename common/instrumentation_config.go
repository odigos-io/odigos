package common

type OptionByContainer struct {
	ContainerName string `json:"containerName"`
	OptionKey     string `json:"optionKey"`
	SpanKind      string `json:"spanKind,omitempty"`
}
