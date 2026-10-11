package helm

import (
	"bytes"
	"embed"
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

//go:embed embedded/*
var embeddedCharts embed.FS

func LoadEmbeddedChart(version string, chartBasename string) (*chart.Chart, error) {
	file := fmt.Sprintf("embedded/%s-%s.tgz", chartBasename, version)
	data, err := embeddedCharts.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("embedded chart %q not found for version %s: %w", chartBasename, version, err)
	}
	return loader.LoadArchive(bytes.NewReader(data))
}
