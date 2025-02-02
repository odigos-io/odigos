module github.com/odigos-io/odigos/distros

require (
	github.com/odigos-io/odigos/common v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common

go 1.23.0

toolchain go1.23.5
