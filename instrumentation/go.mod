module github.com/odigos-io/odigos/instrumentation

go 1.24.0

require (
	github.com/go-logr/logr v1.4.3
	github.com/odigos-io/odigos/distros v0.0.0
	github.com/odigos-io/runtime-detector v0.0.8
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/metric v1.35.0
	golang.org/x/sync v0.14.0
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/oklog/run v1.1.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
)

require (
	github.com/cilium/ebpf v0.19.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/hashicorp/go-plugin v1.6.3
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/odigos-io/odigos/common v0.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common

replace github.com/odigos-io/odigos/distros => ../distros
