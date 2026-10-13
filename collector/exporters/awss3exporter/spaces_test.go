// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awss3exporter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestScopedCredentialsTLSRetryAndOTLPRoundTrip(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	keys := map[string]bool{}
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "GET" {
			_, _ = w.Write(objects[r.URL.Path])
			return
		}
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Credential=spaces-key/")
		assert.Empty(t, r.Header.Get("X-Amz-Sdk-Checksum-Algorithm"))
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`<Error><Code>SlowDown</Code></Error>`))
			return
		}
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NotContains(t, keys, r.URL.Path)
		keys[r.URL.Path] = true
		objects[r.URL.Path] = payload
		w.Header().Set("ETag", `"test-etag"`)
	}))
	defer server.Close()
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	certPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(certPath, cert, 0600))
	t.Setenv("AWS_CA_BUNDLE", certPath)
	t.Setenv("AWS_ACCESS_KEY_ID", "unrelated-global-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "unrelated-global-secret")

	for _, format := range []MarshalerType{OtlpJSON, OtlpProtobuf} {
		for _, compression := range []configcompression.Type{"", configcompression.TypeGzip} {
			for _, signal := range []string{"traces", "metrics", "logs"} {
				t.Run(string(format)+"/"+string(compression)+"/"+signal, func(t *testing.T) {
					cfg := createDefaultConfig().(*Config)
					cfg.MarshalerName = format
					cfg.S3Uploader.Endpoint = server.URL
					cfg.S3Uploader.S3Bucket = "telemetry"
					cfg.S3Uploader.S3ForcePathStyle = true
					cfg.S3Uploader.AccessKeyID = configopaque.String("spaces-key")
					cfg.S3Uploader.SecretAccessKey = configopaque.String("spaces-secret")
					cfg.S3Uploader.ChecksumWhenRequired = true
					cfg.S3Uploader.UniqueKeyFuncName = "uuidv7"
					cfg.S3Uploader.Compression = compression
					require.NoError(t, cfg.Validate())
					e := newS3Exporter(cfg, signal, exporter.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}})
					require.NoError(t, e.start(context.Background(), nil))
					switch signal {
					case "traces":
						data := ptrace.NewTraces()
						for _, service := range []string{"checkout", "inventory"} {
							rs := data.ResourceSpans().AppendEmpty()
							rs.Resource().Attributes().PutStr("service.name", service)
							rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("operation")
						}
						require.NoError(t, e.ConsumeTraces(context.Background(), data))
					case "metrics":
						data := pmetric.NewMetrics()
						for _, service := range []string{"checkout", "inventory"} {
							rm := data.ResourceMetrics().AppendEmpty()
							rm.Resource().Attributes().PutStr("service.name", service)
							m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
							m.SetName("requests")
							m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
						}
						require.NoError(t, e.ConsumeMetrics(context.Background(), data))
					case "logs":
						data := plog.NewLogs()
						for _, service := range []string{"checkout", "inventory"} {
							rl := data.ResourceLogs().AppendEmpty()
							rl.Resource().Attributes().PutStr("service.name", service)
							rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("request complete")
						}
						require.NoError(t, e.ConsumeLogs(context.Background(), data))
					}
					mu.Lock()
					var key string
					for k := range objects {
						if strings.Contains(k, "/"+signal+"_") {
							key = k
						}
					}
					mu.Unlock()
					require.NotEmpty(t, key)
					res, err := server.Client().Get(server.URL + key)
					require.NoError(t, err)
					raw, err := io.ReadAll(res.Body)
					require.NoError(t, err)
					require.NoError(t, res.Body.Close())
					if compression == configcompression.TypeGzip {
						reader, err := gzip.NewReader(bytes.NewReader(raw))
						require.NoError(t, err)
						raw, err = io.ReadAll(reader)
						require.NoError(t, err)
						require.NoError(t, reader.Close())
					}
					switch signal {
					case "traces":
						var u ptrace.Unmarshaler = &ptrace.JSONUnmarshaler{}
						if format == OtlpProtobuf {
							u = &ptrace.ProtoUnmarshaler{}
						}
						data, err := u.UnmarshalTraces(raw)
						require.NoError(t, err)
						require.Equal(t, 2, data.SpanCount())
						require.Equal(t, 2, data.ResourceSpans().Len())
					case "metrics":
						var u pmetric.Unmarshaler = &pmetric.JSONUnmarshaler{}
						if format == OtlpProtobuf {
							u = &pmetric.ProtoUnmarshaler{}
						}
						data, err := u.UnmarshalMetrics(raw)
						require.NoError(t, err)
						require.Equal(t, 2, data.DataPointCount())
						require.Equal(t, 2, data.ResourceMetrics().Len())
					case "logs":
						var u plog.Unmarshaler = &plog.JSONUnmarshaler{}
						if format == OtlpProtobuf {
							u = &plog.ProtoUnmarshaler{}
						}
						data, err := u.UnmarshalLogs(raw)
						require.NoError(t, err)
						require.Equal(t, 2, data.LogRecordCount())
						require.Equal(t, 2, data.ResourceLogs().Len())
					}
					mu.Lock()
					delete(objects, key)
					mu.Unlock()
				})
			}
		}
	}
	require.Equal(t, 13, attempts)
}
