module github.com/odigos-io/odigos/procdiscovery

go 1.26.1

require github.com/odigos-io/odigos/common v0.0.0

require github.com/hashicorp/go-version v1.8.0 // indirect

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	sigs.k8s.io/controller-runtime v0.23.3 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
