module github.com/odigos-io/odigos/distros

require (
	github.com/hashicorp/go-version v1.8.0
	github.com/odigos-io/odigos/common v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/odigos-io/odigos/common => ../common

go 1.26.1
