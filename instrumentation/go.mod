module github.com/odigos-io/odigos/instrumentation

go 1.24.0

require (
	github.com/go-logr/logr v1.4.3
	github.com/odigos-io/odigos/distros v0.0.0
	github.com/odigos-io/runtime-detector v0.0.14
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/metric v1.37.0
	golang.org/x/sync v0.16.0
)

require (
	github.com/cilium/ebpf v0.19.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/odigos-io/odigos/common v0.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common

replace github.com/odigos-io/odigos/distros => ../distros
