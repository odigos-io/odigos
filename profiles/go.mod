module github.com/odigos-io/odigos/profiles

go 1.26.1

require github.com/odigos-io/odigos/common v0.0.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
