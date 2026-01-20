module github.com/odigos-io/odigos/instrumentation

go 1.25.0

require (
	github.com/cilium/ebpf v0.19.1-0.20250815145053-c9de60689836
	github.com/go-logr/logr v1.4.3
	github.com/odigos-io/odigos/common v0.0.0
	github.com/odigos-io/odigos/distros v0.0.0
	github.com/odigos-io/runtime-detector v0.0.22
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/metric v1.38.0
	golang.org/x/sync v0.17.0
)

require (
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common

replace github.com/odigos-io/odigos/distros => ../distros
