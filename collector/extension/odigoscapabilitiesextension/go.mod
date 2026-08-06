module github.com/odigos-io/odigos/collector/extension/odigoscapabilitiesextension

go 1.26.2

require (
	github.com/odigos-io/odigos/common v0.0.0
	go.opentelemetry.io/collector/component v1.57.0
	go.opentelemetry.io/collector/extension v1.57.0
	go.uber.org/zap v1.28.0
)

replace github.com/odigos-io/odigos/common => ../../../common

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	go.opentelemetry.io/collector/featuregate v1.57.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.151.0 // indirect
	go.opentelemetry.io/collector/pdata v1.57.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)
