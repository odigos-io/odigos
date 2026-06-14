package serviceioconnector

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:   "empty config is valid",
			config: Config{},
		},
		{
			name: "valid attributes",
			config: Config{
				InputSpanAttributes:  []string{"http.route"},
				OutputSpanAttributes: []string{"rpc.service"},
			},
		},
		{
			name: "empty input attribute",
			config: Config{
				InputSpanAttributes: []string{"http.route", "  "},
			},
			wantErr: "input_span_attributes[1] must not be empty",
		},
		{
			name: "duplicate output attribute",
			config: Config{
				OutputSpanAttributes: []string{"http.route", "http.route"},
			},
			wantErr: "output_span_attributes contains duplicate key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)
	factories.Connectors[component.MustNewType("serviceio")] = NewFactory()

	cfg, err := otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "serviceio-connector-config.yaml"), factories)
	require.NoError(t, err)

	connectorCfg := cfg.Connectors[component.NewID(component.MustNewType("serviceio"))].(*Config)
	require.Equal(t, []string{"http.route", "rpc.method"}, connectorCfg.InputSpanAttributes)
	require.Equal(t, []string{"http.route", "rpc.service", "db.system"}, connectorCfg.OutputSpanAttributes)
	require.Equal(t, 30*time.Second, *connectorCfg.MetricsFlushInterval)
}

func TestMetadataConfigUnmarshal(t *testing.T) {
	factory := NewFactory()
	cm, err := confmaptest.LoadConf("metadata.yaml")
	require.NoError(t, err)

	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub("tests::config")
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(&cfg))

	typedCfg := cfg.(*Config)
	require.Equal(t, []string{"http.route"}, typedCfg.InputSpanAttributes)
	require.Equal(t, []string{"rpc.service"}, typedCfg.OutputSpanAttributes)
}
