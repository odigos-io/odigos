// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awss3exporter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestSwiftArchiveRoundTrip requires a real Swift proxy with s3api/s3token and
// Keystone EC2 credentials. It is deliberately not a mock S3 acceptance test.
func TestSwiftArchiveRoundTrip(t *testing.T) {
	endpoint := os.Getenv("SWIFT_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set SWIFT_TEST_ENDPOINT, SWIFT_TEST_ACCESS_KEY and SWIFT_TEST_SECRET_KEY for a deployed Swift S3 gateway")
	}
	access, secret := os.Getenv("SWIFT_TEST_ACCESS_KEY"), os.Getenv("SWIFT_TEST_SECRET_KEY")
	require.NotEmpty(t, access)
	require.NotEmpty(t, secret)
	ctx := t.Context()
	client := s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String(endpoint), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""), RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired})
	bucket := "odigos-swift-compatibility"
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &bucket})
	if err != nil {
		_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &bucket})
		require.NoError(t, err)
	}
	target, err := url.Parse(endpoint)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	var rejected, uploads, parts atomic.Int32
	var rejectCompletion atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			uploads.Add(1)
			if rejectCompletion.Load() && r.Method == http.MethodPost && r.URL.Query().Get("uploadId") != "" {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, "<Error><Code>AccessDenied</Code></Error>")
				return
			}
			if r.URL.Query().Get("partNumber") != "" {
				parts.Add(1)
			}
			if rejected.CompareAndSwap(0, 1) {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, "<Error><Code>ServiceUnavailable</Code></Error>")
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
	defer server.Close()
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
	configFor := func(prefix string) *Config {
		cfg := createDefaultConfig().(*Config)
		cfg.S3Uploader.S3Bucket = bucket
		cfg.S3Uploader.Endpoint = server.URL
		cfg.S3Uploader.S3ForcePathStyle = true
		cfg.S3Uploader.UseStaticCredentials = true
		cfg.S3Uploader.AccessKeyID = configopaque.String(access)
		cfg.S3Uploader.SecretAccessKey = configopaque.String(secret)
		cfg.S3Uploader.RequestChecksumCalculation = "when_required"
		cfg.S3Uploader.S3Prefix = prefix
		cfg.S3Uploader.TLS = &configtls.ClientConfig{Config: configtls.Config{CAPem: configopaque.String(ca)}}
		return cfg
	}
	type archiveObject struct {
		key  string
		body []byte
	}
	getObjects := func(prefix string) []archiveObject {
		result, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &prefix})
		require.NoError(t, err)
		var objects []archiveObject
		for _, obj := range result.Contents {
			out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: obj.Key})
			require.NoError(t, err)
			b, err := io.ReadAll(out.Body)
			require.NoError(t, err)
			require.NoError(t, out.Body.Close())
			objects = append(objects, archiveObject{key: *obj.Key, body: b})
		}
		return objects
	}
	for _, format := range []MarshalerType{OtlpJSON, OtlpProtobuf} {
		for _, compression := range []configcompression.Type{"", configcompression.TypeGzip, configcompression.TypeZstd} {
			t.Run(string(format)+"/"+string(compression), func(t *testing.T) {
				prefix := fmt.Sprintf("roundtrip/%d/", time.Now().UnixNano())
				cfg := configFor(prefix)
				cfg.MarshalerName = format
				cfg.S3Uploader.Compression = compression
				traces, metrics, logs := ptrace.NewTraces(), pmetric.NewMetrics(), plog.NewLogs()
				for _, service := range []string{"frontend", "inventory"} {
					tr := traces.ResourceSpans().AppendEmpty()
					tr.Resource().Attributes().PutStr("service.name", service)
					tr.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("GET /buy")
					mr := metrics.ResourceMetrics().AppendEmpty()
					mr.Resource().Attributes().PutStr("service.name", service)
					metric := mr.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
					metric.SetName("requests")
					metric.SetEmptySum().DataPoints().AppendEmpty().SetIntValue(42)
					extra := mr.ScopeMetrics().At(0).Metrics()
					gauge := extra.AppendEmpty()
					gauge.SetName("temperature")
					gauge.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(18.5)
					histogram := extra.AppendEmpty()
					histogram.SetName("latency")
					hp := histogram.SetEmptyHistogram().DataPoints().AppendEmpty()
					hp.SetCount(2)
					hp.SetSum(3)
					hp.BucketCounts().FromRaw([]uint64{1, 1})
					hp.ExplicitBounds().FromRaw([]float64{1})
					exponential := extra.AppendEmpty()
					exponential.SetName("exponential_latency")
					ep := exponential.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
					ep.SetCount(1)
					ep.SetSum(1)
					ep.Positive().BucketCounts().FromRaw([]uint64{1})
					summary := extra.AppendEmpty()
					summary.SetName("summary_latency")
					sp := summary.SetEmptySummary().DataPoints().AppendEmpty()
					sp.SetCount(1)
					sp.SetSum(1)
					lr := logs.ResourceLogs().AppendEmpty()
					lr.Resource().Attributes().PutStr("service.name", service)
					lr.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("request completed")
				}
				for _, signal := range []string{"traces", "metrics", "logs"} {
					exp := newS3Exporter(cfg, signal, exportertest.NewNopSettings(NewFactory().Type()))
					require.NoError(t, exp.start(ctx, nil))
					switch signal {
					case "traces":
						err = exp.ConsumeTraces(ctx, traces)
					case "metrics":
						err = exp.ConsumeMetrics(ctx, metrics)
					case "logs":
						err = exp.ConsumeLogs(ctx, logs)
					}
					require.NoError(t, err)
				}
				objects := getObjects(prefix)
				require.Len(t, objects, 3)
				counts := map[string]int{}
				expected := map[string][]byte{}
				expected["traces"], err = (&ptrace.JSONMarshaler{}).MarshalTraces(traces)
				require.NoError(t, err)
				expected["metrics"], err = (&pmetric.JSONMarshaler{}).MarshalMetrics(metrics)
				require.NoError(t, err)
				expected["logs"], err = (&plog.JSONMarshaler{}).MarshalLogs(logs)
				require.NoError(t, err)
				for _, object := range objects {
					b := object.body
					if compression == configcompression.TypeGzip {
						r, e := gzip.NewReader(bytes.NewReader(b))
						require.NoError(t, e)
						b, e = io.ReadAll(r)
						require.NoError(t, e)
						require.NoError(t, r.Close())
					}
					if compression == configcompression.TypeZstd {
						r, e := zstd.NewReader(nil)
						require.NoError(t, e)
						b, e = r.DecodeAll(b, nil)
						require.NoError(t, e)
						r.Close()
					}
					if format == OtlpJSON {
						switch {
						case bytes.Contains(b, []byte("resourceSpans")):
							v, e := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(b)
							require.NoError(t, e)
							counts["traces"] = v.ResourceSpans().Len()
							actual, marshalErr := (&ptrace.JSONMarshaler{}).MarshalTraces(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["traces"]), string(actual))
						case bytes.Contains(b, []byte("resourceMetrics")):
							v, e := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(b)
							require.NoError(t, e)
							counts["metrics"] = v.ResourceMetrics().Len()
							actual, marshalErr := (&pmetric.JSONMarshaler{}).MarshalMetrics(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["metrics"]), string(actual))
						case bytes.Contains(b, []byte("resourceLogs")):
							v, e := (&plog.JSONUnmarshaler{}).UnmarshalLogs(b)
							require.NoError(t, e)
							counts["logs"] = v.ResourceLogs().Len()
							actual, marshalErr := (&plog.JSONMarshaler{}).MarshalLogs(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["logs"]), string(actual))
						}
					} else {
						// The S3 filename identifies the signal; protobuf envelopes otherwise share field numbers.
						key := object.key
						switch {
						case strings.Contains(key, "traces_"):
							v, e := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
							require.NoError(t, e)
							counts["traces"] = v.ResourceSpans().Len()
							actual, marshalErr := (&ptrace.JSONMarshaler{}).MarshalTraces(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["traces"]), string(actual))
						case strings.Contains(key, "metrics_"):
							v, e := (&pmetric.ProtoUnmarshaler{}).UnmarshalMetrics(b)
							require.NoError(t, e)
							counts["metrics"] = v.ResourceMetrics().Len()
							actual, marshalErr := (&pmetric.JSONMarshaler{}).MarshalMetrics(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["metrics"]), string(actual))
						case strings.Contains(key, "logs_"):
							v, e := (&plog.ProtoUnmarshaler{}).UnmarshalLogs(b)
							require.NoError(t, e)
							counts["logs"] = v.ResourceLogs().Len()
							actual, marshalErr := (&plog.JSONMarshaler{}).MarshalLogs(v)
							require.NoError(t, marshalErr)
							require.JSONEq(t, string(expected["logs"]), string(actual))
						}
					}
				}
				require.Equal(t, map[string]int{"traces": 2, "metrics": 2, "logs": 2}, counts)
			})
		}
	}
	t.Run("multipart", func(t *testing.T) {
		prefix := fmt.Sprintf("multipart/%d/", time.Now().UnixNano())
		cfg := configFor(prefix)
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(strings.Repeat("Swift multipart telemetry ", 1000000))
		exp := newS3Exporter(cfg, "logs", exportertest.NewNopSettings(NewFactory().Type()))
		require.NoError(t, exp.start(ctx, nil))
		require.NoError(t, exp.ConsumeLogs(ctx, data))
		objects := getObjects(prefix)
		require.Len(t, objects, 1)
		decoded, e := (&plog.JSONUnmarshaler{}).UnmarshalLogs(objects[0].body)
		require.NoError(t, e)
		require.Equal(t, data.LogRecordCount(), decoded.LogRecordCount())
		require.Equal(t, data.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str(), decoded.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str())
		require.Positive(t, parts.Load())
	})
	t.Run("multipart_completion_rejection", func(t *testing.T) {
		rejectCompletion.Store(true)
		defer rejectCompletion.Store(false)
		prefix := fmt.Sprintf("incomplete/%d/", time.Now().UnixNano())
		cfg := configFor(prefix)
		cfg.S3Uploader.RetryMaxAttempts = 1
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(strings.Repeat("must not partially archive ", 1000000))
		exp := newS3Exporter(cfg, "logs", exportertest.NewNopSettings(NewFactory().Type()))
		require.NoError(t, exp.start(ctx, nil))
		require.Error(t, exp.ConsumeLogs(ctx, data))
		require.Empty(t, getObjects(prefix))
	})
	t.Run("missing_bucket_rejection", func(t *testing.T) {
		cfg := configFor("missing/")
		cfg.S3Uploader.S3Bucket = "odigos-bucket-that-does-not-exist"
		cfg.S3Uploader.RetryMaxAttempts = 1
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("must not archive")
		exp := newS3Exporter(cfg, "logs", exportertest.NewNopSettings(NewFactory().Type()))
		require.NoError(t, exp.start(ctx, nil))
		require.Error(t, exp.ConsumeLogs(ctx, data))
	})

	t.Run("authentication_rejection", func(t *testing.T) {
		cfg := configFor("rejected/")
		cfg.S3Uploader.SecretAccessKey = "wrong-secret"
		cfg.S3Uploader.RetryMaxAttempts = 1
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("must not archive")
		exp := newS3Exporter(cfg, "logs", exportertest.NewNopSettings(NewFactory().Type()))
		require.NoError(t, exp.start(ctx, nil))
		require.Error(t, exp.ConsumeLogs(context.Background(), data))
		require.Empty(t, getObjects("rejected/"))
	})
	t.Run("untrusted_TLS_rejection", func(t *testing.T) {
		cfg := configFor("untrusted/")
		cfg.S3Uploader.TLS = nil
		cfg.S3Uploader.RetryMaxAttempts = 1
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("must not archive")
		exp := newS3Exporter(cfg, "logs", exportertest.NewNopSettings(NewFactory().Type()))
		require.NoError(t, exp.start(ctx, nil))
		require.Error(t, exp.ConsumeLogs(ctx, data))
		require.Empty(t, getObjects("untrusted/"))
	})
	require.Equal(t, int32(1), rejected.Load())
	require.Greater(t, uploads.Load(), int32(19))
}
