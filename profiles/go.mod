module github.com/odigos-io/odigos/profiles

go 1.24.0

require github.com/odigos-io/odigos/common v0.0.0

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
