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

// LoadEmbeddedChart loads the odigos chart with the given version (e.g. "1.3.2")
// from the embedded filesystem.
func LoadEmbeddedChart(version string) (*chart.Chart, error) {
	file := fmt.Sprintf("embedded/odigos-%s.tgz", version)
	data, err := embeddedCharts.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("embedded chart not found for version %s: %w", version, err)
	}
	return loader.LoadArchive(bytes.NewReader(data))
}
