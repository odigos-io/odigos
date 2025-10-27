module github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/azureblobstorageexporter

go 1.25.0

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.4.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.44.0
	go.opentelemetry.io/collector/component/componenttest v0.138.0
	go.opentelemetry.io/collector/confmap v1.44.0
	go.opentelemetry.io/collector/consumer v1.44.0
	go.opentelemetry.io/collector/exporter v1.44.0
	go.opentelemetry.io/collector/exporter/exporterhelper v0.138.0
	go.opentelemetry.io/collector/exporter/exportertest v0.138.0
	go.opentelemetry.io/collector/pdata v1.44.0
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.13.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/client v1.44.0 // indirect
	go.opentelemetry.io/collector/config/configoptional v1.44.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.44.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.138.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.138.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.138.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.138.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.138.0 // indirect
	go.opentelemetry.io/collector/extension v1.44.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.138.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.44.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.138.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.138.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.138.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.44.0 // indirect
	go.opentelemetry.io/collector/receiver v1.44.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.138.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.138.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.13.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/log v0.14.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/grpc v1.76.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
