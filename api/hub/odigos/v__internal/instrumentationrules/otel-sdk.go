package instrumentationrules

import "github.com/odigos-io/odigos/common"

type OtelSdks struct {
	OtelSdkByLanguage map[common.ProgrammingLanguage]common.OtelSdk `json:"otelSdkByLanguage"`
}

type OtelDistros struct {
	OtelDistroNames []string `json:"otelDistroNames"`
}
