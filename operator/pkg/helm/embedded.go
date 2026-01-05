/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// OdigosChartVersion is injected at build time via -ldflags
var OdigosChartVersion string

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

