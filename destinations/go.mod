module github.com/odigos-io/odigos/destinations

go 1.22.0

require (
	github.com/odigos-io/odigos/common v1.0.48
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
)

replace github.com/odigos-io/odigos/common => ../common
