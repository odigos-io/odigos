module github.com/odigos-io/odigos/collector/receivers/ebpfprofilerwrapper

go 1.26.2

require (
	github.com/odigos-io/odigos/common v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/component v1.57.0
	go.opentelemetry.io/collector/consumer/xconsumer v0.151.0
	go.opentelemetry.io/collector/receiver v1.57.0
	go.opentelemetry.io/collector/receiver/xreceiver v0.151.0
	go.opentelemetry.io/ebpf-profiler v0.0.202614
	go.uber.org/zap v1.28.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.21.0 // indirect
	github.com/elastic/go-freelru v0.16.0 // indirect
	github.com/elastic/go-perf v0.0.0-20260224073651-af0ee0c731b7 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/mdlayher/kobject v0.0.0-20200520190114-19ca17470d7d // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/collector/consumer v1.57.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.57.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.151.0 // indirect
	go.opentelemetry.io/collector/pdata v1.57.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.151.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.57.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/arch v0.25.0 // indirect
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	sigs.k8s.io/controller-runtime v0.23.3 // indirect
)

replace github.com/odigos-io/odigos/common => ../../../common
