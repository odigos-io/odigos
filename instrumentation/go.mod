module github.com/odigos-io/odigos/instrumentation

go 1.24.0

require (
	github.com/go-logr/logr v1.4.2
	github.com/odigos-io/odigos/common v0.0.0
	github.com/odigos-io/runtime-detector v0.0.7
	go.opentelemetry.io/otel v1.35.0
	golang.org/x/sync v0.12.0
)

require (
	github.com/cilium/ebpf v0.17.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
