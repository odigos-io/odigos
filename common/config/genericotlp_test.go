package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestGrpcOAuth2AutoTLS(t *testing.T) {
	tests := []struct {
		name            string
		config          map[string]string
		expectedTLS     bool
		expectedAuth    bool
		expectedExtName string
		expectTLSConfig bool // New field to check if TLS config should be present
	}{
		{
			name: "OAuth2 enabled forces TLS when TLS was disabled",
			config: map[string]string{
				"OTLP_GRPC_ENDPOINT":         "example.com:4317",
				"OTLP_GRPC_TLS_ENABLED":      "false", // User disabled TLS
				"OTLP_GRPC_OAUTH2_ENABLED":   "true",  // But OAuth2 is enabled
				"OTLP_GRPC_OAUTH2_CLIENT_ID": "test-client-id",
				"OTLP_GRPC_OAUTH2_TOKEN_URL": "https://auth.example.com/token",
			},
			expectedTLS:     true, // TLS should be forced to true
			expectedAuth:    true,
			expectedExtName: "oauth2client/otlpgrpc-test-id",
			expectTLSConfig: true, // TLS config should be present
		},
		{
			name: "OAuth2 enabled with TLS already enabled",
			config: map[string]string{
				"OTLP_GRPC_ENDPOINT":         "example.com:4317",
				"OTLP_GRPC_TLS_ENABLED":      "true", // TLS explicitly enabled
				"OTLP_GRPC_OAUTH2_ENABLED":   "true",
				"OTLP_GRPC_OAUTH2_CLIENT_ID": "test-client-id",
				"OTLP_GRPC_OAUTH2_TOKEN_URL": "https://auth.example.com/token",
			},
			expectedTLS:     true,
			expectedAuth:    true,
			expectedExtName: "oauth2client/otlpgrpc-test-id",
			expectTLSConfig: true, // TLS config should be present
		},
		{
			name: "TLS enabled without OAuth2",
			config: map[string]string{
				"OTLP_GRPC_ENDPOINT":    "example.com:4317",
				"OTLP_GRPC_TLS_ENABLED": "true",
			},
			expectedTLS:     true,
			expectedAuth:    false,
			expectTLSConfig: true, // TLS config should be present
		},
		{
			name: "Neither TLS nor OAuth2 enabled - no TLS config",
			config: map[string]string{
				"OTLP_GRPC_ENDPOINT": "example.com:4317",
				// Neither TLS nor OAuth2 enabled
			},
			expectedTLS:     false,
			expectedAuth:    false,
			expectTLSConfig: true, // TLS config should be present with insecure=true
		},
		{
			name: "OAuth2 disabled, TLS setting respected",
			config: map[string]string{
				"OTLP_GRPC_ENDPOINT":       "example.com:4317",
				"OTLP_GRPC_TLS_ENABLED":    "false",
				"OTLP_GRPC_OAUTH2_ENABLED": "false",
			},
			expectedTLS:     false,
			expectedAuth:    false,
			expectTLSConfig: true, // TLS config should be present with insecure=true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock destination
			dest := &mockGrpcDestination{
				id:     "test-id",
				config: tt.config,
			}

			// Create Generic OTLP configurator
			genericOtlp := &GenericOTLP{}
			
			// Create initial config
			config := &Config{
				Extensions: make(map[string]interface{}),
				Exporters:  make(map[string]interface{}),
				Service: Service{
					Extensions: []string{},
					Pipelines:  make(map[string]Pipeline),
				},
			}

			// Apply the configuration
			pipelineNames, err := genericOtlp.ModifyConfig(dest, config)

			assert.NoError(t, err)
			assert.NotEmpty(t, pipelineNames)

			// Check exporter configuration
			exporterName := "otlp/generic-test-id"
			assert.Contains(t, config.Exporters, exporterName)
			exporterConfig := config.Exporters[exporterName].(GenericMap)
			
			// Check TLS configuration presence
			if tt.expectTLSConfig {
				assert.Contains(t, exporterConfig, "tls", "TLS config should be present")
				tlsConfig := exporterConfig["tls"].(GenericMap)
				
				if tt.expectedTLS {
					assert.False(t, tlsConfig["insecure"].(bool), "TLS should be enabled (insecure=false)")
				} else {
					assert.True(t, tlsConfig["insecure"].(bool), "TLS should be disabled (insecure=true)")
				}
			} else {
				assert.NotContains(t, exporterConfig, "tls", "TLS config should NOT be present when neither TLS nor OAuth2 are enabled")
			}

			// Check OAuth2 configuration
			if tt.expectedAuth {
				assert.Contains(t, config.Service.Extensions, tt.expectedExtName)
				assert.Contains(t, config.Extensions, tt.expectedExtName)
				
				// Verify exporter has auth configuration
				authConfig := exporterConfig["auth"].(GenericMap)
				assert.Equal(t, tt.expectedExtName, authConfig["authenticator"])
			} else {
				// Should not have OAuth2 extension
				assert.NotContains(t, config.Service.Extensions, "oauth2client/otlpgrpc-test-id")
				assert.NotContains(t, config.Extensions, "oauth2client/otlpgrpc-test-id")
				assert.Nil(t, exporterConfig["auth"])
			}
		})
	}
}

// Mock destination for gRPC testing
type mockGrpcDestination struct {
	id     string
	config map[string]string
}

func (m *mockGrpcDestination) GetID() string {
	return m.id
}

func (m *mockGrpcDestination) GetConfig() map[string]string {
	return m.config
}

func (m *mockGrpcDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}

func (m *mockGrpcDestination) GetType() common.DestinationType {
	return common.GenericOTLPDestinationType
}