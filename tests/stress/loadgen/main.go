package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	SpansPerSecond    int     `json:"spans_per_second"`
	Duration          string  `json:"duration"`
	TraceComplexity   string  `json:"trace_complexity"`
	Pattern           string  `json:"pattern"`
	AttributesPerSpan int     `json:"attributes_per_span"`
	SpansPerTrace     int     `json:"spans_per_trace"`
	LargeAttributes   bool    `json:"large_attributes"`
	BurstDuration     string  `json:"burst_duration"`
	QuietDuration     string  `json:"quiet_duration"`
	Endpoint          string  `json:"endpoint"`
	ServiceName       string  `json:"service_name"`
	TestID            string  `json:"test_id"`
}

type LoadGenerator struct {
	config     *Config
	tracer     trace.Tracer
	shutdown   func(context.Context) error
	stats      *Stats
	ctx        context.Context
	cancel     context.CancelFunc
}

type Stats struct {
	mu               sync.RWMutex
	SpansGenerated   int64     `json:"spans_generated"`
	TracesGenerated  int64     `json:"traces_generated"`
	Errors           int64     `json:"errors"`
	StartTime        time.Time `json:"start_time"`
	LastReportTime   time.Time `json:"last_report_time"`
	BytesGenerated   int64     `json:"bytes_generated"`
	SpansPerSecond   float64   `json:"current_spans_per_second"`
}

func main() {
	var configFile = flag.String("config", "", "Configuration file path")
	var endpoint = flag.String("endpoint", "http://localhost:4317", "OTLP gRPC endpoint")
	var spansPerSec = flag.Int("spans-per-sec", 1000, "Spans per second")
	var duration = flag.String("duration", "10m", "Test duration")
	var serviceName = flag.String("service-name", "stress-loadgen", "Service name")
	var testID = flag.String("test-id", "default", "Test ID")
	flag.Parse()

	var config *Config
	if *configFile != "" {
		config = loadConfigFromFile(*configFile)
	} else {
		config = &Config{
			SpansPerSecond:    *spansPerSec,
			Duration:          *duration,
			TraceComplexity:   "simple",
			Pattern:           "constant",
			AttributesPerSpan: 5,
			SpansPerTrace:     3,
			Endpoint:          *endpoint,
			ServiceName:       *serviceName,
			TestID:            *testID,
		}
	}

	log.Printf("Starting load generator with config: %+v", config)

	lg, err := NewLoadGenerator(config)
	if err != nil {
		log.Fatalf("Failed to create load generator: %v", err)
	}
	defer lg.Shutdown()

	// Start statistics reporting
	go lg.reportStats()

	// Parse duration
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		log.Fatalf("Invalid duration: %v", err)
	}

	log.Printf("Running load test for %v", duration)
	
	// Start load generation
	lg.Run(duration)
	
	log.Printf("Load test completed. Final stats: %+v", lg.GetStats())
}

func loadConfigFromFile(filename string) *Config {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return &config
}

func NewLoadGenerator(config *Config) (*LoadGenerator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create OTLP exporter
	conn, err := grpc.DialContext(ctx, config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("test.id", config.TestID),
			attribute.String("test.type", "stress"),
		),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(1*time.Second),
			sdktrace.WithMaxExportBatchSize(1000),
		),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("stress-loadgen")

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	stats := &Stats{
		StartTime:      time.Now(),
		LastReportTime: time.Now(),
	}

	return &LoadGenerator{
		config:   config,
		tracer:   tracer,
		shutdown: shutdown,
		stats:    stats,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (lg *LoadGenerator) Run(duration time.Duration) {
	endTime := time.Now().Add(duration)
	
	switch lg.config.Pattern {
	case "constant":
		lg.runConstantLoad(endTime)
	case "burst":
		lg.runBurstLoad(endTime)
	case "ramp-up":
		lg.runRampUpLoad(endTime)
	default:
		lg.runConstantLoad(endTime)
	}
}

func (lg *LoadGenerator) runConstantLoad(endTime time.Time) {
	ticker := time.NewTicker(time.Second / time.Duration(lg.config.SpansPerSecond))
	defer ticker.Stop()

	for {
		select {
		case <-lg.ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(endTime) {
				return
			}
			lg.generateTrace()
		}
	}
}

func (lg *LoadGenerator) runBurstLoad(endTime time.Time) {
	burstDuration, _ := time.ParseDuration(lg.config.BurstDuration)
	quietDuration, _ := time.ParseDuration(lg.config.QuietDuration)
	
	if burstDuration == 0 {
		burstDuration = 2 * time.Minute
	}
	if quietDuration == 0 {
		quietDuration = 1 * time.Minute
	}

	for time.Now().Before(endTime) {
		// Burst period
		log.Printf("Starting burst period for %v", burstDuration)
		burstEnd := time.Now().Add(burstDuration)
		ticker := time.NewTicker(time.Second / time.Duration(lg.config.SpansPerSecond))
		
		for time.Now().Before(burstEnd) && time.Now().Before(endTime) {
			select {
			case <-lg.ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				lg.generateTrace()
			}
		}
		ticker.Stop()

		// Quiet period
		if time.Now().Before(endTime) {
			log.Printf("Starting quiet period for %v", quietDuration)
			time.Sleep(quietDuration)
		}
	}
}

func (lg *LoadGenerator) runRampUpLoad(endTime time.Time) {
	// TODO: Implement ramp-up pattern
	lg.runConstantLoad(endTime)
}

func (lg *LoadGenerator) generateTrace() {
	traceCtx, span := lg.tracer.Start(lg.ctx, "root-span")
	defer span.End()

	// Add attributes based on configuration
	lg.addSpanAttributes(span)

	// Generate child spans
	for i := 0; i < lg.config.SpansPerTrace-1; i++ {
		childCtx, childSpan := lg.tracer.Start(traceCtx, fmt.Sprintf("child-span-%d", i))
		lg.addSpanAttributes(childSpan) 
		
		// Simulate some work
		time.Sleep(time.Microsecond * time.Duration(i*10))
		
		childSpan.End()
	}

	lg.updateStats()
}

func (lg *LoadGenerator) addSpanAttributes(span trace.Span) {
	baseAttrs := []attribute.KeyValue{
		attribute.String("service.name", lg.config.ServiceName),
		attribute.String("test.id", lg.config.TestID),
		attribute.Int("test.spans_per_sec", lg.config.SpansPerSecond),
		attribute.String("test.pattern", lg.config.Pattern),
	}

	span.SetAttributes(baseAttrs...)

	// Add complexity-based attributes
	switch lg.config.TraceComplexity {
	case "simple":
		lg.addSimpleAttributes(span)
	case "complex":
		lg.addComplexAttributes(span)
	default:
		lg.addSimpleAttributes(span)
	}
}

func (lg *LoadGenerator) addSimpleAttributes(span trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", "GET"),
		attribute.String("http.url", "/api/test"),
		attribute.Int("http.status_code", 200),
		attribute.String("component", "http-client"),
		attribute.Bool("success", true),
	}

	// Add additional attributes up to the configured count
	for i := len(attrs); i < lg.config.AttributesPerSpan; i++ {
		attrs = append(attrs, attribute.String(fmt.Sprintf("attr.%d", i), fmt.Sprintf("value-%d", i)))
	}

	span.SetAttributes(attrs...)
}

func (lg *LoadGenerator) addComplexAttributes(span trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/api/complex/endpoint"),
		attribute.Int("http.status_code", 201),
		attribute.String("http.user_agent", "StressTest/1.0"),
		attribute.String("http.remote_addr", "192.168.1.100"),
		attribute.String("database.type", "postgresql"),
		attribute.String("database.connection_string", "postgres://user:pass@db:5432/testdb"),
		attribute.String("message.type", "application/json"),
		attribute.Int("message.size", 1024),
		attribute.String("user.id", generateRandomID()),
		attribute.String("session.id", generateRandomID()),
		attribute.String("request.id", generateRandomID()),
		attribute.Float64("duration.ms", float64(time.Now().UnixNano()%10000)/1000),
		attribute.Bool("cache.hit", time.Now().UnixNano()%2 == 0),
		attribute.String("region", "us-west-2"),
	}

	// Add large attributes if configured
	if lg.config.LargeAttributes {
		largeValue := strings.Repeat("x", 1024) // 1KB attribute
		attrs = append(attrs, attribute.String("large.data", largeValue))
		
		// Add JSON-like attribute
		jsonLikeValue := fmt.Sprintf(`{"timestamp":%d,"data":"%s","nested":{"field1":"value1","field2":"value2"}}`, 
			time.Now().UnixNano(), strings.Repeat("data", 100))
		attrs = append(attrs, attribute.String("json.payload", jsonLikeValue))
	}

	// Add additional random attributes
	for i := len(attrs); i < lg.config.AttributesPerSpan; i++ {
		attrs = append(attrs, attribute.String(fmt.Sprintf("complex.attr.%d", i), 
			fmt.Sprintf("complex-value-%s-%d", generateRandomID(), i)))
	}

	span.SetAttributes(attrs...)
}

func generateRandomID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (lg *LoadGenerator) updateStats() {
	lg.stats.mu.Lock()
	defer lg.stats.mu.Unlock()
	
	lg.stats.SpansGenerated += int64(lg.config.SpansPerTrace)
	lg.stats.TracesGenerated++
	
	// Estimate bytes generated (rough calculation)
	bytesPerSpan := 500 // Base size
	if lg.config.LargeAttributes {
		bytesPerSpan += 2048 // Additional for large attributes
	}
	bytesPerSpan += lg.config.AttributesPerSpan * 50 // Rough attribute size
	
	lg.stats.BytesGenerated += int64(bytesPerSpan * lg.config.SpansPerTrace)
}

func (lg *LoadGenerator) reportStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lg.ctx.Done():
			return
		case <-ticker.C:
			stats := lg.GetStats()
			
			elapsed := time.Since(stats.LastReportTime)
			if elapsed > 0 {
				stats.SpansPerSecond = float64(stats.SpansGenerated) / time.Since(stats.StartTime).Seconds()
			}
			
			log.Printf("Stats: Spans=%d, Traces=%d, Errors=%d, Rate=%.2f spans/sec, Bytes=%d MB",
				stats.SpansGenerated, stats.TracesGenerated, stats.Errors,
				stats.SpansPerSecond, stats.BytesGenerated/1024/1024)
			
			lg.stats.mu.Lock()
			lg.stats.LastReportTime = time.Now()
			lg.stats.mu.Unlock()
		}
	}
}

func (lg *LoadGenerator) GetStats() Stats {
	lg.stats.mu.RLock()
	defer lg.stats.mu.RUnlock()
	return *lg.stats
}

func (lg *LoadGenerator) Shutdown() {
	lg.cancel()
	if lg.shutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		lg.shutdown(ctx)
	}
}