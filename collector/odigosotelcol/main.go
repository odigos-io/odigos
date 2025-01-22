// Code generated by "go.opentelemetry.io/collector/cmd/builder". DO NOT EDIT.

// Program odigosotelcol is an OpenTelemetry Collector binary.
package main

import (
	"log"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	odigosfileprovider "go.opentelemetry.io/collector/odigos/providers/odigosfileprovider"
	envprovider "go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/otelcol"
)

func main() {
	info := component.BuildInfo{
		Command:     "odigosotelcol",
		Description: "OpenTelemetry Collector for Odigos",
		Version:     "0.118.0",
	}

	set := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				ProviderFactories: []confmap.ProviderFactory{
					odigosfileprovider.NewFactory(),
					envprovider.NewFactory(),
				},
			},
		},
	}

	if err := run(set); err != nil {
		log.Fatal(err)
	}
}

func runInteractive(params otelcol.CollectorSettings) error {
	cmd := otelcol.NewCommand(params)
	if err := cmd.Execute(); err != nil {
		log.Fatalf("collector server run finished with error: %v", err)
	}

	return nil
}
