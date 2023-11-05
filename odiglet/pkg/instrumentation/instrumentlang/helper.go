package instrumentlang

import (
	"os"
)

func getAdditionalResourceAttributes() string {
	otelResourceAttributes, exists := os.LookupEnv("OTEL_RESOURCE_ATTRIBUTES")
	if exists {
		return otelResourceAttributes
	}
	return ""
}
