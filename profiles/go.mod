module github.com/odigos-io/odigos/profiles

go 1.26.2

require (
	github.com/ianlancetaylor/demangle v0.0.0-20240312041847-bd984b5ce465
	github.com/odigos-io/odigos/common v0.0.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
