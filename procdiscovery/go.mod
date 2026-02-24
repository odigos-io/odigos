module github.com/odigos-io/odigos/procdiscovery

go 1.26.0

require github.com/odigos-io/odigos/common v0.0.0

require github.com/hashicorp/go-version v1.7.0 // indirect

require (
	github.com/go-logr/logr v1.4.3
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
