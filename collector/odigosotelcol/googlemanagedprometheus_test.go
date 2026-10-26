package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	gmpexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/odigos-io/odigos/common"
	odigosconfig "github.com/odigos-io/odigos/common/config"
)

type gmpTestDestination struct{ data map[string]string }

func (g gmpTestDestination) GetID() string { return "test" }
func (g gmpTestDestination) GetType() common.DestinationType {
	return common.GoogleManagedPrometheusDestinationType
}
func (g gmpTestDestination) GetConfig() map[string]string { return g.data }
func (g gmpTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.MetricsObservabilitySignal}
}

type gmpTestServer struct {
	monitoringpb.UnimplementedMetricServiceServer
	mu            sync.Mutex
	requests      []*monitoringpb.CreateTimeSeriesRequest
	code          codes.Code
	retry         bool
	compression   string
	authorization string
}

func (s *gmpTestServer) CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	md, _ := metadata.FromIncomingContext(ctx)
	s.authorization = strings.Join(md.Get("authorization"), "")
	if s.retry && len(s.requests) == 1 {
		return nil, status.Error(codes.Unavailable, "temporary outage")
	}
	if s.code != codes.OK {
		st, _ := status.New(s.code, "test rejection").WithDetails(&monitoringpb.CreateTimeSeriesSummary{TotalPointCount: int32(len(req.TimeSeries)), SuccessPointCount: 1})
		return nil, st.Err()
	}
	return &emptypb.Empty{}, nil
}
func (s *gmpTestServer) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (s *gmpTestServer) HandleRPC(_ context.Context, st stats.RPCStats) {
	if h, ok := st.(*stats.InHeader); ok {
		s.mu.Lock()
		s.compression = h.Compression
		s.mu.Unlock()
	}
}
func (s *gmpTestServer) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}
func (s *gmpTestServer) HandleConn(context.Context, stats.ConnStats) {}

func gmpPipeline(t *testing.T, customize func(*gmpexporter.Config)) consumer.Metrics {
	t.Helper()
	ctx := context.Background()
	cfg := odigosconfig.Config{Exporters: odigosconfig.GenericMap{}, Processors: odigosconfig.GenericMap{}, Service: odigosconfig.Service{Pipelines: map[string]odigosconfig.Pipeline{}}}
	names, err := (&odigosconfig.GoogleManagedPrometheus{}).ModifyConfig(gmpTestDestination{map[string]string{"GCP_PROJECT_ID": "test-project", "GMP_CLUSTER": "core1777-e2e", "GMP_LOCATION": "us-central1", "GCP_TIMEOUT": "10s"}}, &cfg)
	require.NoError(t, err)
	factories, err := components()
	require.NoError(t, err)
	pipeline := cfg.Service.Pipelines[names[0]]
	ef := factories.Exporters[component.MustNewType("googlemanagedprometheus")]
	ecfg := ef.CreateDefaultConfig().(*gmpexporter.Config)
	require.NoError(t, gmpConfMap(t, cfg.Exporters[pipeline.Exporters[0]]).Unmarshal(ecfg))
	require.NoError(t, ecfg.Validate())
	customize(ecfg)
	// Consume synchronously so API rejection is observable in this test.
	ecfg.QueueSettings = configoptional.None[exporterhelper.QueueBatchConfig]()
	exp, err := ef.CreateMetrics(ctx, exportertest.NewNopSettings(ef.Type()), ecfg)
	require.NoError(t, err)
	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })
	var next consumer.Metrics = exp
	for i := len(pipeline.Processors) - 1; i >= 0; i-- {
		name := pipeline.Processors[i]
		typ := component.MustNewType(strings.SplitN(name, "/", 2)[0])
		pf := factories.Processors[typ]
		pcfg := pf.CreateDefaultConfig()
		require.NoError(t, gmpConfMap(t, cfg.Processors[name]).Unmarshal(pcfg))
		proc, err := pf.CreateMetrics(ctx, processortest.NewNopSettings(typ), pcfg, next)
		require.NoError(t, err)
		require.NoError(t, proc.Start(ctx, componenttest.NewNopHost()))
		t.Cleanup(func() { require.NoError(t, proc.Shutdown(ctx)) })
		next = proc
	}
	return next
}

func gmpMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	for _, pid := range []int64{101, 102} {
		rm := md.ResourceMetrics().AppendEmpty()
		attrs := rm.Resource().Attributes()
		attrs.PutStr("service.name", "odigos-test")
		attrs.PutStr("host.name", "test-vm")
		attrs.PutInt("process.pid", pid)
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("core1777")
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("odigos.test.requests")
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp := sum.DataPoints().AppendEmpty()
		dp.SetIntValue(7)
		dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Minute)))
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		for _, key := range []string{"location", "cluster", "namespace", "job", "instance", "project_id"} {
			dp.Attributes().PutStr(key, "original-"+key)
		}
		histogram := sm.Metrics().AppendEmpty()
		histogram.SetName("odigos.test.duration")
		histogram.SetUnit("s")
		h := histogram.SetEmptyHistogram()
		h.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		hp := h.DataPoints().AppendEmpty()
		hp.SetCount(2)
		hp.SetSum(0.7)
		hp.ExplicitBounds().FromRaw([]float64{0.5})
		hp.BucketCounts().FromRaw([]uint64{1, 1})
		hp.SetStartTimestamp(dp.StartTimestamp())
		hp.SetTimestamp(dp.Timestamp())
		ex := hp.Exemplars().AppendEmpty()
		ex.SetDoubleValue(0.6)
		ex.SetTimestamp(dp.Timestamp())
		ex.SetTraceID(pcommon.TraceID{1})
		ex.SetSpanID(pcommon.SpanID{2})
	}
	return md
}

func TestGoogleManagedPrometheusTLSConversion(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	cert := certServer.TLS.Certificates[0]
	certServer.Close()
	roots := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	roots.AddCert(parsed)
	for _, tc := range []struct {
		name  string
		code  codes.Code
		retry bool
	}{
		{name: "success"}, {name: "transient-error", code: codes.Unavailable, retry: true}, {name: "partial-rejection", code: codes.InvalidArgument}, {name: "unauthenticated", code: codes.Unauthenticated}, {name: "permission-denied", code: codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &gmpTestServer{code: tc.code, retry: tc.retry}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})), grpc.StatsHandler(server))
			monitoringpb.RegisterMetricServiceServer(grpcServer, server)
			go grpcServer.Serve(listener)
			t.Cleanup(grpcServer.Stop)
			pipeline := gmpPipeline(t, func(cfg *gmpexporter.Config) {
				cfg.GMPConfig.MetricConfig.ClientConfig.Endpoint = listener.Addr().String()
				cfg.GMPConfig.MetricConfig.ClientConfig.GetClientOptions = func() []option.ClientOption {
					return []option.ClientOption{option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})), option.WithGRPCDialOption(grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12})))}
				}
			})
			err = pipeline.ConsumeMetrics(context.Background(), gmpMetrics())
			if tc.code == codes.OK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.code.String())
			}
			server.mu.Lock()
			defer server.mu.Unlock()
			require.Equal(t, "gzip", server.compression)
			require.Equal(t, "Bearer test-token", server.authorization)
			require.NotEmpty(t, server.requests)
			if tc.retry {
				require.Len(t, server.requests, 1)
			}
			instances := map[string]bool{}
			histograms := 0
			for _, ts := range server.requests[len(server.requests)-1].TimeSeries {
				require.Equal(t, "prometheus_target", ts.Resource.Type)
				require.Equal(t, "us-central1", ts.Resource.Labels["location"])
				require.Equal(t, "core1777-e2e", ts.Resource.Labels["cluster"])
				if strings.Contains(ts.Metric.Type, "odigos_test_requests") {
					instances[ts.Resource.Labels["instance"]] = true
					for _, key := range []string{"location", "cluster", "namespace", "job", "instance", "project_id"} {
						require.Equal(t, "original-"+key, ts.Metric.Labels["exported_"+key])
						require.NotContains(t, ts.Metric.Labels, key)
					}
				}
				if dist := ts.Points[0].Value.GetDistributionValue(); dist != nil {
					histograms++
					require.EqualValues(t, 2, dist.Count)
					require.Len(t, dist.Exemplars, 1)
					require.NotEmpty(t, dist.Exemplars[0].Attachments)
				}
			}
			require.Equal(t, map[string]bool{"test-vm/101": true, "test-vm/102": true}, instances)
			require.Equal(t, 2, histograms)
		})
	}
}

func gmpConfMap(t *testing.T, value any) *confmap.Conf {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(encoded, &data))
	return confmap.NewFromStringMap(data)
}
