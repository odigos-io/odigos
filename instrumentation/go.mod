module github.com/odigos-io/odigos/instrumentation

go 1.23.0

require (
	github.com/go-logr/logr v1.4.2
	github.com/odigos-io/odigos/common v0.0.0
	github.com/odigos-io/runtime-detector v0.0.6
	go.opentelemetry.io/otel v1.34.0
	golang.org/x/sync v0.10.0
)

require (
	github.com/cilium/ebpf v0.16.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
