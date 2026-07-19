module github.com/odigos-io/odigos/instrumentation

go 1.26.2

require (
	github.com/cilium/ebpf v0.20.0
	github.com/go-logr/logr v1.4.3
	github.com/odigos-io/odigos/common v0.0.0
	github.com/odigos-io/odigos/distros v0.0.0
	github.com/odigos-io/runtime-detector v0.0.25
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/metric v1.44.0
	golang.org/x/sync v0.20.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/jsimonetti/rtnetlink/v2 v2.0.3 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	sigs.k8s.io/controller-runtime v0.24.1 // indirect
)

replace github.com/odigos-io/odigos/common => ../common

replace github.com/odigos-io/odigos/distros => ../distros
