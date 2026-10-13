package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestFirehoseCloudWatchBridge(t *testing.T) {
	for _, scenario := range []string{"multi-resource", "retry", "rejected-subset", "invalid-auth", "untrusted-tls"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
			t.Setenv("AWS_SESSION_TOKEN", "test-session")
			t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
			var mu sync.Mutex
			calls := 0
			var messages []map[string]any
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				require.NotNil(t, r.TLS)
				require.Contains(t, r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=test-key/")
				require.Contains(t, r.Header.Get("Authorization"), "/us-east-1/logs/aws4_request")
				require.Equal(t, "test-session", r.Header.Get("X-Amz-Security-Token"))
				if strings.HasSuffix(r.Header.Get("X-Amz-Target"), ".CreateLogStream") {
					w.Header().Set("Content-Type", "application/x-amz-json-1.1")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"__type":"ResourceAlreadyExistsException","message":"precreated stream"}`))
					return
				}
				require.True(t, strings.HasSuffix(r.Header.Get("X-Amz-Target"), ".PutLogEvents"))
				var input struct {
					LogGroupName  string `json:"logGroupName"`
					LogStreamName string `json:"logStreamName"`
					Events        []struct {
						Message string `json:"message"`
					} `json:"logEvents"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
				require.Equal(t, "/odigos/firehose", input.LogGroupName)
				require.Equal(t, "odigos", input.LogStreamName)
				calls++
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")
				if scenario == "retry" && calls == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"__type":"ServiceUnavailableException","message":"retry"}`))
					return
				}
				if scenario == "invalid-auth" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"__type":"UnrecognizedClientException","message":"invalid credentials"}`))
					return
				}
				for _, event := range input.Events {
					var msg map[string]any
					require.NoError(t, json.Unmarshal([]byte(event.Message), &msg))
					messages = append(messages, msg)
				}
				if scenario == "rejected-subset" {
					_, _ = w.Write([]byte(`{"nextSequenceToken":"unused","rejectedLogEventsInfo":{"tooOldLogEventEndIndex":0}}`))
				} else {
					_, _ = w.Write([]byte(`{"nextSequenceToken":"unused"}`))
				}
			}))
			defer server.Close()

			factories, err := components()
			require.NoError(t, err)
			factory := factories.Exporters[component.MustNewType("awscloudwatchlogs")]
			require.NotNil(t, factory)
			cfg := factory.CreateDefaultConfig()
			require.NoError(t, confmap.NewFromStringMap(map[string]any{
				"log_group_name":  "/odigos/firehose",
				"log_stream_name": "odigos",
				"region":          "us-east-1",
				"raw_log":         false,
				"endpoint":        server.URL,
				"no_verify_ssl":   scenario != "untrusted-tls",
				"sending_queue":   map[string]any{"enabled": false},
				"retry_on_failure": map[string]any{
					"enabled":          scenario == "retry",
					"initial_interval": "1ms", "max_interval": "10ms", "max_elapsed_time": "5s",
				},
			}).Unmarshal(cfg))
			exp, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
			require.NoError(t, err)
			require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
			defer func() { require.NoError(t, exp.Shutdown(context.Background())) }()
			data := plog.NewLogs()
			for _, service := range []string{"frontend", "backend"} {
				resource := data.ResourceLogs().AppendEmpty()
				resource.Resource().Attributes().PutStr("service.name", service)
				scope := resource.ScopeLogs().AppendEmpty()
				scope.Scope().SetName("odigos-e2e")
				record := scope.LogRecords().AppendEmpty()
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
				record.Body().SetStr("firehose-e2e-" + service)
				record.Attributes().PutStr("e2e.marker", "CORE-1870")
			}
			err = exp.ConsumeLogs(context.Background(), data)
			if scenario == "invalid-auth" || scenario == "untrusted-tls" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			mu.Lock()
			defer mu.Unlock()
			require.Len(t, messages, 2, "submitted events must not be replayed after a successful response")
			if scenario == "retry" {
				require.Equal(t, 2, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			for i, service := range []string{"frontend", "backend"} {
				require.Equal(t, "firehose-e2e-"+service, messages[i]["body"])
				require.Equal(t, service, messages[i]["resource"].(map[string]any)["service.name"])
				require.Equal(t, "CORE-1870", messages[i]["attributes"].(map[string]any)["e2e.marker"])
				require.Equal(t, "odigos-e2e", messages[i]["scope"].(map[string]any)["name"])
			}
		})
	}
}
