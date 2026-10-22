package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	pb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	pubsub "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

type pubSubDestination struct{ data map[string]string }

func (d pubSubDestination) GetID() string { return "pubsub" }
func (d pubSubDestination) GetType() common.DestinationType {
	return common.GoogleCloudPubSubDestinationType
}
func (d pubSubDestination) GetConfig() map[string]string { return d.data }
func (d pubSubDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}
}

type pubSubPublisher struct {
	pb.UnimplementedPublisherServer
	mu       sync.Mutex
	calls    int
	failAt   int
	reject   codes.Code
	messages []*pb.PubsubMessage
}

func (p *pubSubPublisher) Publish(_ context.Context, request *pb.PublishRequest) (*pb.PublishResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.reject != codes.OK {
		return nil, status.Error(p.reject, "injected rejection")
	}
	if p.calls == p.failAt {
		return nil, status.Error(codes.PermissionDenied, "injected transient failure")
	}
	if proto.Size(request) > 10_000_000 {
		return nil, status.Error(codes.InvalidArgument, "publish request exceeds 10 MB")
	}
	if request.Topic != "projects/test-project/topics/otlp" {
		return nil, status.Error(codes.NotFound, "topic not found")
	}
	p.messages = append(p.messages, request.Messages...)
	return &pb.PublishResponse{MessageIds: []string{fmt.Sprint(p.calls)}}, nil
}
func (p *pubSubPublisher) snapshot() []*pb.PubsubMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*pb.PubsubMessage(nil), p.messages...)
}

func pubSubConfig(t *testing.T, compression, ordering string, publisher *pubSubPublisher) *pubsub.Config {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.MaxRecvMsgSize(12_000_000))
	pb.RegisterPublisherServer(server, publisher)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	cfg := &config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	_, err = (&config.GoogleCloudPubSub{}).ModifyConfig(pubSubDestination{data: map[string]string{
		"GCP_PUBSUB_TOPIC":        "projects/test-project/topics/otlp",
		"GCP_PUBSUB_COMPRESSION":  compression,
		"GCP_PUBSUB_ORDERING_KEY": ordering,
	}}, cfg)
	require.NoError(t, err)
	factories, err := components()
	require.NoError(t, err)
	factory := factories.Exporters[component.MustNewType("googlecloudpubsub")]
	require.NotNil(t, factory)
	actual := factory.CreateDefaultConfig().(*pubsub.Config)
	require.NoError(t, confmap.NewFromStringMap(map[string]any(cfg.Exporters["googlecloudpubsub/pubsub"].(config.GenericMap))).Unmarshal(actual))
	require.NoError(t, actual.Validate())
	require.False(t, actual.Insecure)
	actual.Endpoint = listener.Addr().String()
	// Only the in-process publisher is plaintext; production destination config retains TLS and ADC.
	actual.Insecure = true
	actual.QueueSettings = configoptional.None[exporterhelper.QueueBatchConfig]()
	actual.BackOffConfig.Enabled = publisher.failAt > 0
	actual.BackOffConfig.InitialInterval = time.Millisecond
	actual.BackOffConfig.MaxInterval = 2 * time.Millisecond
	actual.BackOffConfig.MaxElapsedTime = time.Second
	return actual
}

func consumePubSubSignal(t *testing.T, cfg *pubsub.Config, signal string, large bool) error {
	t.Helper()
	factories, err := components()
	require.NoError(t, err)
	factory := factories.Exporters[component.MustNewType("googlecloudpubsub")]
	settings := exportertest.NewNopSettings(factory.Type())
	ctx := t.Context()
	host := componenttest.NewNopHost()
	switch signal {
	case "traces":
		exp, err := factory.CreateTraces(ctx, settings, cfg)
		require.NoError(t, err)
		require.NoError(t, exp.Start(ctx, host))
		defer func() { require.NoError(t, exp.Shutdown(ctx)) }()
		data := ptrace.NewTraces()
		for _, name := range []string{"frontend", "backend"} {
			resource := data.ResourceSpans().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", name)
			span := resource.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
			span.SetName("GET /checkout")
			span.SetStartTimestamp(1)
			span.SetEndTimestamp(2)
		}
		return exp.ConsumeTraces(ctx, data)
	case "metrics":
		exp, err := factory.CreateMetrics(ctx, settings, cfg)
		require.NoError(t, err)
		require.NoError(t, exp.Start(ctx, host))
		defer func() { require.NoError(t, exp.Shutdown(ctx)) }()
		data := pmetric.NewMetrics()
		for _, name := range []string{"frontend", "backend"} {
			resource := data.ResourceMetrics().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", name)
			metric := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
			metric.SetName("http.server.requests")
			point := metric.SetEmptyGauge().DataPoints().AppendEmpty()
			point.SetIntValue(42)
			point.SetTimestamp(1)
		}
		return exp.ConsumeMetrics(ctx, data)
	default:
		exp, err := factory.CreateLogs(ctx, settings, cfg)
		require.NoError(t, err)
		require.NoError(t, exp.Start(ctx, host))
		defer func() { require.NoError(t, exp.Shutdown(ctx)) }()
		data := plog.NewLogs()
		for _, name := range []string{"frontend", "backend"} {
			resource := data.ResourceLogs().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", name)
			log := resource.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			log.SetTimestamp(1)
			log.Body().SetStr("checkout completed")
			if large {
				log.Body().SetStr(strings.Repeat("x", 5_100_000))
			}
		}
		return exp.ConsumeLogs(ctx, data)
	}
}

func TestGoogleCloudPubSubWire(t *testing.T) {
	for _, signal := range []string{"traces", "metrics", "logs"} {
		for _, compression := range []string{"none", "gzip"} {
			for _, ordering := range []string{"", "service.name"} {
				for _, failAt := range []int{0, 1, 2} {
					t.Run(fmt.Sprintf("%s/%s/ordering=%s/fail=%d", signal, compression, ordering, failAt), func(t *testing.T) {
						if ordering == "" && failAt == 2 {
							t.Skip("unordered batch makes one publish call")
						}
						publisher := &pubSubPublisher{failAt: failAt}
						cfg := pubSubConfig(t, compression, ordering, publisher)
						require.NoError(t, consumePubSubSignal(t, cfg, signal, false))
						seen := map[string]int{}
						for _, message := range publisher.snapshot() {
							attrs := message.Attributes
							require.Equal(t, "org.opentelemetry.otlp."+signal+".v1", attrs["ce-type"])
							require.Equal(t, "application/protobuf", attrs["content-type"])
							require.Equal(t, "1.0", attrs["ce-specversion"])
							require.NotEmpty(t, attrs["ce-id"])
							require.NotEmpty(t, attrs["ce-source"])
							_, err := time.Parse(time.RFC3339Nano, attrs["ce-time"])
							require.NoError(t, err)
							payload := message.Data
							if compression == "gzip" {
								require.Equal(t, "gzip", attrs["content-encoding"])
								reader, err := gzip.NewReader(bytes.NewReader(payload))
								require.NoError(t, err)
								payload, err = io.ReadAll(reader)
								require.NoError(t, err)
								require.NoError(t, reader.Close())
							} else {
								require.Empty(t, attrs["content-encoding"])
							}
							switch signal {
							case "traces":
								data, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(payload)
								require.NoError(t, err)
								for i := 0; i < data.ResourceSpans().Len(); i++ {
									r := data.ResourceSpans().At(i)
									name, _ := r.Resource().Attributes().Get("service.name")
									seen[name.Str()]++
									require.Equal(t, "GET /checkout", r.ScopeSpans().At(0).Spans().At(0).Name())
									if ordering != "" {
										require.Equal(t, name.Str(), message.OrderingKey)
									}
								}
							case "metrics":
								data, err := (&pmetric.ProtoUnmarshaler{}).UnmarshalMetrics(payload)
								require.NoError(t, err)
								for i := 0; i < data.ResourceMetrics().Len(); i++ {
									r := data.ResourceMetrics().At(i)
									name, _ := r.Resource().Attributes().Get("service.name")
									seen[name.Str()]++
									require.Equal(t, int64(42), r.ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).IntValue())
									if ordering != "" {
										require.Equal(t, name.Str(), message.OrderingKey)
									}
								}
							default:
								data, err := (&plog.ProtoUnmarshaler{}).UnmarshalLogs(payload)
								require.NoError(t, err)
								for i := 0; i < data.ResourceLogs().Len(); i++ {
									r := data.ResourceLogs().At(i)
									name, _ := r.Resource().Attributes().Get("service.name")
									seen[name.Str()]++
									require.Equal(t, "checkout completed", r.ScopeLogs().At(0).LogRecords().At(0).Body().Str())
									if ordering != "" {
										require.Equal(t, name.Str(), message.OrderingKey)
									}
								}
							}
							if ordering == "" {
								require.Empty(t, message.OrderingKey)
							}
						}
						require.Positive(t, seen["frontend"])
						require.Positive(t, seen["backend"])
						if ordering != "" && failAt == 2 {
							require.Equal(t, 3, seen["frontend"]+seen["backend"], "partial publish retry can duplicate the already accepted resource")
						}
					})
				}
			}
		}
	}
}

func TestGoogleCloudPubSubRejection(t *testing.T) {
	for _, code := range []codes.Code{codes.PermissionDenied, codes.NotFound, codes.InvalidArgument} {
		t.Run(code.String(), func(t *testing.T) {
			publisher := &pubSubPublisher{reject: code}
			cfg := pubSubConfig(t, "none", "", publisher)
			require.Error(t, consumePubSubSignal(t, cfg, "logs", false))
			require.Empty(t, publisher.snapshot())
		})
	}
	publisher := &pubSubPublisher{}
	cfg := pubSubConfig(t, "none", "", publisher)
	require.Error(t, consumePubSubSignal(t, cfg, "logs", true), "oversized batches must be rejected")
	require.Empty(t, publisher.snapshot())
}
