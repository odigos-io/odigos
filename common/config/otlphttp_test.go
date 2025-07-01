package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func TestParseOtlpHttpEndpoint(t *testing.T) {
	type args struct {
		rawURL string
		port   string
		path   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid url with http scheme",
			args: args{
				rawURL: "http://localhost:4318",
				port:   "4318",
				path:   "",
			},
			want:    "http://localhost:4318",
			wantErr: false,
		},
		{
			name: "valid url with https scheme",
			args: args{
				rawURL: "https://localhost:4318",
				port:   "4318",
				path:   "",
			},
			want:    "https://localhost:4318",
			wantErr: false,
		},
		{
			name: "invalid scheme",
			args: args{
				rawURL: "invalid://localhost:4318",
				port:   "4318",
				path:   "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "path allowed",
			args: args{
				rawURL: "http://localhost:4318/some-path",
				port:   "4318",
				path:   "/some-path",
			},
			want:    "http://localhost:4318/some-path",
			wantErr: false,
		},
		{
			name: "path mismatch not allowed",
			args: args{
				rawURL: "http://localhost:4318/some-path",
				port:   "4318",
				path:   "/some-other-path",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "path in URL allowed",
			args: args{
				rawURL: "http://localhost:4318/some-path",
				port:   "4318",
				path:   "",
			},
			want:    "http://localhost:4318/some-path",
			wantErr: false,
		},
		{
			name: "ipv4",
			args: args{
				rawURL: "http://127.0.0.1:4318",
				port:   "4318",
				path:   "",
			},
			want:    "http://127.0.0.1:4318",
			wantErr: false,
		},
		{
			name: "ipv6",
			args: args{
				rawURL: "http://[::1]:4318",
				port:   "4318",
				path:   "",
			},
			want:    "http://[::1]:4318",
			wantErr: false,
		},
		{
			name: "do not add port when missing",
			args: args{
				rawURL: "http://localhost",
				port:   "",
				path:   "",
			},
			want:    "http://localhost",
			wantErr: false,
		},
		{
			name: "do not add port when missing with ipv6",
			args: args{
				rawURL: "http://[::1]",
				port:   "",
				path:   "",
			},
			want:    "http://[::1]",
			wantErr: false,
		},
		{
			name: "host with dots",
			args: args{
				rawURL: "http://jaeger.tracing:4318",
				port:   "4318",
				path:   "",
			},
			want:    "http://jaeger.tracing:4318",
			wantErr: false,
		},
		{
			name: "non numeric port",
			args: args{
				rawURL: "http://localhost:port",
				port:   "",
				path:   "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non numeric port with ipv6",
			args: args{
				rawURL: "http://[::1]:port",
				port:   "",
				path:   "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "default port",
			args: args{
				rawURL: "http://localhost",
				port:   "1234",
				path:   "",
			},
			want:    "http://localhost:1234",
			wantErr: false,
		},
		{
			name: "non default port",
			args: args{
				rawURL: "http://localhost:1234",
				port:   "1234",
				path:   "",
			},
			want:    "http://localhost:1234",
			wantErr: false,
		},
		{
			name: "default port missmatched",
			args: args{
				rawURL: "http://localhost:1234",
				port:   "1111",
				path:   "",
			},
			want:    "http://localhost:1234",
			wantErr: false,
		},
		{
			name: "whitespaces",
			args: args{
				rawURL: "  http://localhost:4318  ",
				port:   "4318",
				path:   "",
			},
			want:    "http://localhost:4318",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOtlpHttpEndpoint(tt.args.rawURL, tt.args.port, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOtlpHttpEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseOtlpHttpEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOAuth2Configuration(t *testing.T) {
	tests := []struct {
		name            string
		config          map[string]string
		expectedError   bool
		expectedAuth    bool
		expectedExtName string
		expectedScopes  []string
		expectTLSConfig bool
	}{
		{
			name: "OAuth2 enabled with all parameters",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":         "https://example.com:4318",
				"OTLP_HTTP_OAUTH2_ENABLED":   "true",
				"OTLP_HTTP_OAUTH2_CLIENT_ID": "test-client-id",
				"OTLP_HTTP_OAUTH2_TOKEN_URL": "https://auth.example.com/token",
				"OTLP_HTTP_OAUTH2_SCOPES":    "api.metrics,api.traces",
				"OTLP_HTTP_OAUTH2_AUDIENCE":  "api.example.com",
				// Note: OTLP_HTTP_OAUTH2_CLIENT_SECRET is handled separately through secrets
			},
			expectedAuth:    true,
			expectedExtName: "oauth2client/otlphttp-test-id",
			expectedScopes:  []string{"api.metrics", "api.traces"},
			expectTLSConfig: true,
		},
		{
			name: "Real user configuration",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":             "http://eden.com",
				"OTLP_HTTP_COMPRESSION":          "none",
				"OTLP_HTTP_TLS_ENABLED":          "false",
				"OTLP_HTTP_INSECURE_SKIP_VERIFY": "false",
				"OTLP_HTTP_OAUTH2_ENABLED":       "true",
				"OTLP_HTTP_OAUTH2_CLIENT_ID":     "123123",
				"OTLP_HTTP_OAUTH2_TOKEN_URL":     "https://gooogle.com",
				"OTLP_HTTP_OAUTH2_SCOPES":        "asdasd",
				"OTLP_HTTP_OAUTH2_AUDIENCE":      "ccccx",
				// Client secret stored separately in secret
			},
			expectedAuth:    true,
			expectedExtName: "oauth2client/otlphttp-test-id",
			expectedScopes:  []string{"asdasd"},
			expectTLSConfig: true,
		},
		{
			name: "TLS enabled without authentication",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":    "https://example.com:4318",
				"OTLP_HTTP_TLS_ENABLED": "true",
			},
			expectedAuth:    false,
			expectTLSConfig: true,
		},
		{
			name: "Basic Auth without TLS",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":            "https://example.com:4318",
				"OTLP_HTTP_TLS_ENABLED":         "false",
				"OTLP_HTTP_BASIC_AUTH_USERNAME": "user",
				"OTLP_HTTP_BASIC_AUTH_PASSWORD": "pass",
			},
			expectedAuth:    true,
			expectedExtName: "basicauth/otlphttp-test-id",
			expectTLSConfig: true,
		},
		{
			name: "Neither TLS nor authentication - no TLS config",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT": "http://example.com:4318",
				// No TLS, no OAuth2, no Basic Auth
			},
			expectedAuth:    false,
			expectTLSConfig: true,
		},
		{
			name: "OAuth2 disabled",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":       "https://example.com:4318",
				"OTLP_HTTP_OAUTH2_ENABLED": "false",
			},
			expectedAuth:    false,
			expectTLSConfig: true,
		},
		{
			name: "OAuth2 enabled but missing required fields",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":         "https://example.com:4318",
				"OTLP_HTTP_OAUTH2_ENABLED":   "true",
				"OTLP_HTTP_OAUTH2_CLIENT_ID": "test-client-id",
				// Missing TOKEN_URL
			},
			expectedError: true,
		},
		{
			name: "Basic Auth takes precedence when OAuth2 disabled",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":            "https://example.com:4318",
				"OTLP_HTTP_OAUTH2_ENABLED":      "false",
				"OTLP_HTTP_BASIC_AUTH_USERNAME": "user",
				"OTLP_HTTP_BASIC_AUTH_PASSWORD": "pass",
			},
			expectedAuth:    true,
			expectedExtName: "basicauth/otlphttp-test-id",
			expectTLSConfig: true,
		},
		{
			name: "OAuth2 takes precedence over Basic Auth",
			config: map[string]string{
				"OTLP_HTTP_ENDPOINT":            "https://example.com:4318",
				"OTLP_HTTP_OAUTH2_ENABLED":      "true",
				"OTLP_HTTP_OAUTH2_CLIENT_ID":    "test-client-id",
				"OTLP_HTTP_OAUTH2_TOKEN_URL":    "https://auth.example.com/token",
				"OTLP_HTTP_BASIC_AUTH_USERNAME": "user",
				"OTLP_HTTP_BASIC_AUTH_PASSWORD": "pass",
			},
			expectedAuth:    true,
			expectedExtName: "oauth2client/otlphttp-test-id",
			expectTLSConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock destination
			dest := &mockDestination{
				id:     "test-id",
				config: tt.config,
			}

			// Create OTLP HTTP configurator
			otlpHttp := &OTLPHttp{}

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
			pipelineNames, err := otlpHttp.ModifyConfig(dest, config)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			if tt.config["OTLP_HTTP_ENDPOINT"] == "" {
				// Should fail if no endpoint
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, pipelineNames)

			// Check exporter configuration
			exporterName := "otlphttp/generic-test-id"
			assert.Contains(t, config.Exporters, exporterName)
			exporterConfig := config.Exporters[exporterName].(GenericMap)

			// Check TLS configuration presence
			if tt.expectTLSConfig {
				assert.Contains(t, exporterConfig, "tls", "TLS config should be present")
				tlsConfig := exporterConfig["tls"].(GenericMap)
				// When authentication is used without explicit TLS, it should be insecure: true
				// When TLS is explicitly enabled, it should be insecure: false
				userTlsEnabled := tt.config["OTLP_HTTP_TLS_ENABLED"] == "true"
				expectedInsecure := !userTlsEnabled
				assert.Equal(t, expectedInsecure, tlsConfig["insecure"].(bool))
			} else {
				assert.NotContains(t, exporterConfig, "tls", "TLS config should NOT be present when neither TLS nor authentication are enabled")
			}

			// Check if OAuth2 extension is configured correctly
			if tt.expectedAuth {
				assert.Contains(t, config.Service.Extensions, tt.expectedExtName)
				assert.Contains(t, config.Extensions, tt.expectedExtName)

				// Verify the extension configuration
				if tt.expectedExtName == "oauth2client/otlphttp-test-id" {
					extConfig := config.Extensions[tt.expectedExtName].(GenericMap)
					assert.Equal(t, tt.config["OTLP_HTTP_OAUTH2_CLIENT_ID"], extConfig["client_id"])
					assert.Equal(t, "${OTLP_HTTP_OAUTH2_CLIENT_SECRET}", extConfig["client_secret"])
					assert.Equal(t, tt.config["OTLP_HTTP_OAUTH2_TOKEN_URL"], extConfig["token_url"])

					if tt.expectedScopes != nil {
						assert.Equal(t, tt.expectedScopes, extConfig["scopes"])
					}

					if tt.config["OTLP_HTTP_OAUTH2_AUDIENCE"] != "" {
						endpointParams := extConfig["endpoint_params"].(GenericMap)
						assert.Equal(t, tt.config["OTLP_HTTP_OAUTH2_AUDIENCE"], endpointParams["audience"])
					}
				}

				// Verify exporter has auth configuration
				authConfig := exporterConfig["auth"].(GenericMap)
				assert.Equal(t, tt.expectedExtName, authConfig["authenticator"])
			} else {
				// Should not have OAuth2 extension
				assert.NotContains(t, config.Service.Extensions, "oauth2client/otlphttp-test-id")
				assert.NotContains(t, config.Extensions, "oauth2client/otlphttp-test-id")
				assert.Nil(t, exporterConfig["auth"])
			}
		})
	}
}

// Mock destination for testing
type mockDestination struct {
	id     string
	config map[string]string
}

func (m *mockDestination) GetID() string {
	return m.id
}

func (m *mockDestination) GetConfig() map[string]string {
	return m.config
}

func (m *mockDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}

func (m *mockDestination) GetType() common.DestinationType {
	return common.OtlpHttpDestinationType
}
