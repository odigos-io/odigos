module github.com/odigos-io/odigos/procdiscovery

go 1.23.0

require (
	github.com/hashicorp/go-version v1.7.0
	github.com/odigos-io/odigos/common v0.0.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
