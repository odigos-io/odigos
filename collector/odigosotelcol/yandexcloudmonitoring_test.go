package main

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/odigos-io/odigos/common"
	odigosconfig "github.com/odigos-io/odigos/common/config"
)

type yandexTestDestination struct{}

func (yandexTestDestination) GetID() string { return "yandex-test" }
func (yandexTestDestination) GetType() common.DestinationType {
	return common.YandexCloudMonitoringDestinationType
}
func (yandexTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.MetricsObservabilitySignal}
}
func (yandexTestDestination) GetConfig() map[string]string {
	return map[string]string{"YANDEX_CLOUD_MONITORING_WORKSPACE_ID": "mon123", "PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS": `["k8s.namespace.name"]`}
}

type yandexTestHost map[component.ID]component.Component

func (h yandexTestHost) GetExtensions() map[component.ID]component.Component { return h }

func newYandexTestExporter(t *testing.T, handler http.HandlerFunc, trustTLS bool) consumer.Metrics {
	t.Helper()
	ctx := context.Background()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	cfg := &odigosconfig.Config{Exporters: odigosconfig.GenericMap{}, Extensions: odigosconfig.GenericMap{}, Processors: odigosconfig.GenericMap{}, Service: odigosconfig.Service{Pipelines: map[string]odigosconfig.Pipeline{}}}
	_, err := (&odigosconfig.YandexCloudMonitoring{}).ModifyConfig(yandexTestDestination{}, cfg)
	require.NoError(t, err)
	factories, err := components()
	require.NoError(t, err)
	authID := component.MustNewIDWithName("bearertokenauth", "yandexcloudmonitoring-yandex-test")
	authFactory := factories.Extensions[authID.Type()]
	authConfig := authFactory.CreateDefaultConfig()
	authMap := cfg.Extensions[authID.String()].(odigosconfig.GenericMap)
	// Resolve the destination-scoped secret as the env provider would at runtime.
	authMap["token"] = "test-api-key"
	require.NoError(t, confmap.NewFromStringMap(authMap).Unmarshal(authConfig))
	auth, err := authFactory.Create(ctx, extensiontest.NewNopSettings(authID.Type()), authConfig)
	require.NoError(t, err)
	host := yandexTestHost{authID: auth}
	require.NoError(t, auth.Start(ctx, host))
	t.Cleanup(func() { require.NoError(t, auth.Shutdown(ctx)) })
	exporterID := component.MustNewIDWithName("prometheusremotewrite", "yandexcloudmonitoring-yandex-test")
	exporterFactory := factories.Exporters[exporterID.Type()]
	exporterConfig := exporterFactory.CreateDefaultConfig()
	exporterMap := cfg.Exporters[exporterID.String()].(odigosconfig.GenericMap)
	httpConfig := exporterMap["http"].(odigosconfig.GenericMap)
	httpConfig["endpoint"] = server.URL + "/prometheus/workspaces/mon123/api/v1/write"
	if trustTLS {
		caFile := filepath.Join(t.TempDir(), "ca.pem")
		require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
		httpConfig["tls"] = odigosconfig.GenericMap{"ca_file": caFile}
	}
	// Synchronous export makes success and permanent rejection observable to the test.
	exporterMap["remote_write_queue"] = odigosconfig.GenericMap{"enabled": false}
	serialized, err := json.Marshal(exporterMap)
	require.NoError(t, err)
	var exporterSettings map[string]any
	require.NoError(t, json.Unmarshal(serialized, &exporterSettings))
	require.NoError(t, confmap.NewFromStringMap(exporterSettings).Unmarshal(exporterConfig))
	require.NoError(t, exporterConfig.(interface{ Validate() error }).Validate())
	exp, err := exporterFactory.CreateMetrics(ctx, exportertest.NewNopSettings(exporterID.Type()), exporterConfig)
	require.NoError(t, err)
	require.NoError(t, exp.Start(ctx, host))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })
	var next consumer.Metrics = exp
	pipeline := cfg.Service.Pipelines["metrics/yandexcloudmonitoring-yandex-test"]
	for i := len(pipeline.Processors) - 1; i >= 0; i-- {
		var id component.ID
		require.NoError(t, id.UnmarshalText([]byte(pipeline.Processors[i])))
		factory := factories.Processors[id.Type()]
		config := factory.CreateDefaultConfig()
		raw, err := json.Marshal(cfg.Processors[id.String()])
		require.NoError(t, err)
		var settings map[string]any
		require.NoError(t, json.Unmarshal(raw, &settings))
		require.NoError(t, confmap.NewFromStringMap(settings).Unmarshal(config))
		proc, err := factory.CreateMetrics(ctx, processortest.NewNopSettings(id.Type()), config, next)
		require.NoError(t, err)
		require.NoError(t, proc.Start(ctx, host))
		t.Cleanup(func() { require.NoError(t, proc.Shutdown(ctx)) })
		next = proc
	}
	return next
}

func yandexTestMetrics(resources int) pmetric.Metrics {
	data := pmetric.NewMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())
	for i := 0; i < resources; i++ {
		resource := data.ResourceMetrics().AppendEmpty()
		resource.Resource().Attributes().PutStr("service.name", "odigos-yandex-e2e")
		resource.Resource().Attributes().PutStr("service.instance.id", fmt.Sprint(i))
		resource.Resource().Attributes().PutStr("k8s.namespace.name", "demo")
		metrics := resource.ScopeMetrics().AppendEmpty().Metrics()
		gauge := metrics.AppendEmpty()
		gauge.SetName("odigos.cpu.usage")
		point := gauge.SetEmptyGauge().DataPoints().AppendEmpty()
		point.SetTimestamp(now)
		point.SetDoubleValue(42)
		counter := metrics.AppendEmpty()
		counter.SetName("odigos.requests")
		sum := counter.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		sp := sum.DataPoints().AppendEmpty()
		sp.SetTimestamp(now)
		sp.SetIntValue(3)
		delta := metrics.AppendEmpty()
		delta.SetName("unsupported.delta")
		ds := delta.SetEmptySum()
		ds.SetIsMonotonic(true)
		ds.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		ds.DataPoints().AppendEmpty().SetTimestamp(now)
		native := metrics.AppendEmpty()
		native.SetName("unsupported.native")
		nh := native.SetEmptyExponentialHistogram()
		nh.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		nh.DataPoints().AppendEmpty().SetTimestamp(now)
		histogram := metrics.AppendEmpty()
		histogram.SetName("odigos.request.duration")
		histogram.SetUnit("s")
		h := histogram.SetEmptyHistogram()
		h.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		hp := h.DataPoints().AppendEmpty()
		hp.SetTimestamp(now)
		hp.SetCount(3)
		hp.SetSum(1.5)
		hp.ExplicitBounds().FromRaw([]float64{0.1, 1})
		hp.BucketCounts().FromRaw([]uint64{1, 1, 1})
	}
	return data
}

func TestYandexCloudMonitoringRemoteWrite(t *testing.T) {
	var mu sync.Mutex
	attempts, accepted, series := 0, 0, 0
	names := map[string]bool{}
	instances := map[string]bool{}
	exp := newYandexTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		require.Equal(t, "/prometheus/workspaces/mon123/api/v1/write", r.URL.Path)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		require.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		require.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		raw, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		var request prompb.WriteRequest
		require.NoError(t, request.Unmarshal(raw))
		require.Empty(t, request.Metadata)
		require.LessOrEqual(t, len(request.Timeseries), 2000)
		for _, ts := range request.Timeseries {
			require.Empty(t, ts.Histograms)
			labels := map[string]string{}
			for _, label := range ts.Labels {
				labels[label.Name] = label.Value
				if label.Name == "__name__" {
					names[label.Value] = true
				}
				if label.Name == "instance" {
					instances[label.Value] = true
				}
				if label.Name == "k8s_namespace_name" {
					require.Equal(t, "demo", label.Value)
				}
			}
			require.Equal(t, "demo", labels["k8s_namespace_name"])
			require.Equal(t, "odigos-yandex-e2e", labels["job"])
			require.NotEmpty(t, labels["instance"])
		}
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if attempts == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		accepted++
		series += len(request.Timeseries)
		w.WriteHeader(http.StatusNoContent)
	}, true)
	require.NoError(t, exp.ConsumeMetrics(context.Background(), yandexTestMetrics(400)))
	mu.Lock()
	defer mu.Unlock()
	require.Greater(t, accepted, 1, "expanded histogram batches must split")
	require.Equal(t, accepted+2, attempts, "429 and 503 must retry")
	require.Len(t, instances, 400, "resources must remain distinct")
	require.Equal(t, 400*7, series, "gauge, counter and classic histogram per resource")
	require.True(t, names["odigos_cpu_usage"])
	require.True(t, names["odigos_requests_total"])
	require.False(t, names["unsupported_native"])
	require.False(t, names["unsupported_delta_total"])
	require.True(t, names["odigos_request_duration_seconds_bucket"])
	require.False(t, names["target_info"])
}

func TestYandexCloudMonitoringPermanentRejection(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 413} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			attempts := 0
			exp := newYandexTestExporter(t, func(w http.ResponseWriter, r *http.Request) { attempts++; w.WriteHeader(status) }, true)
			err := exp.ConsumeMetrics(context.Background(), yandexTestMetrics(1))
			require.Error(t, err)
			require.True(t, consumererror.IsPermanent(err))
			require.Equal(t, 1, attempts)
		})
	}
}

func TestYandexCloudMonitoringVerifiesTLS(t *testing.T) {
	exp := newYandexTestExporter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("untrusted TLS endpoint must not receive metrics")
	}, false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.Error(t, exp.ConsumeMetrics(ctx, yandexTestMetrics(1)))
}
