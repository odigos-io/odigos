package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	Port               int           `json:"port"`
	Delay              time.Duration `json:"delay"`
	SuccessRate        float64       `json:"success_rate"`
	BackpressureThresh int64         `json:"backpressure_threshold"`
	MaxConcurrency     int           `json:"max_concurrency"`
}

type MockBackend struct {
	config       *Config
	requestCount int64
	activeConns  int64
	totalBytes   int64
	errorCount   int64
	
	// Prometheus metrics
	requestsTotal  prometheus.Counter
	requestLatency prometheus.Histogram
	activeConnGauge prometheus.Gauge
	bytesReceived  prometheus.Counter
	errorsTotal    prometheus.Counter
}

func main() {
	var port = flag.Int("port", 14318, "Server port")
	var delay = flag.Duration("delay", 10*time.Millisecond, "Response delay")
	var successRate = flag.Float64("success-rate", 99.9, "Success rate percentage")
	var backpressureThresh = flag.Int64("backpressure-threshold", 10000, "Backpressure threshold")
	var maxConcurrency = flag.Int("max-concurrency", 1000, "Maximum concurrent connections")
	flag.Parse()

	config := &Config{
		Port:               *port,
		Delay:              *delay,
		SuccessRate:        *successRate,
		BackpressureThresh: *backpressureThresh,
		MaxConcurrency:     *maxConcurrency,
	}

	backend := NewMockBackend(config)
	
	log.Printf("Starting mock backend on port %d with config: %+v", config.Port, config)
	
	if err := backend.Start(); err != nil {
		log.Fatalf("Failed to start backend: %v", err)
	}
}

func NewMockBackend(config *Config) *MockBackend {
	return &MockBackend{
		config: config,
		requestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mock_backend_requests_total",
			Help: "Total number of requests received",
		}),
		requestLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mock_backend_request_duration_seconds",
			Help:    "Request processing duration",
			Buckets: prometheus.DefBuckets,
		}),
		activeConnGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mock_backend_active_connections",
			Help: "Number of active connections",
		}),
		bytesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mock_backend_bytes_received_total",
			Help: "Total bytes received",
		}),
		errorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mock_backend_errors_total",
			Help: "Total number of errors",
		}),
	}
}

func (mb *MockBackend) Start() error {
	// Register Prometheus metrics
	prometheus.MustRegister(mb.requestsTotal)
	prometheus.MustRegister(mb.requestLatency)
	prometheus.MustRegister(mb.activeConnGauge)
	prometheus.MustRegister(mb.bytesReceived)
	prometheus.MustRegister(mb.errorsTotal)

	// Start HTTP server for metrics and health checks
	go mb.startHTTPServer()

	// Start gRPC server for OTLP
	return mb.startGRPCServer()
}

func (mb *MockBackend) startHTTPServer() {
	mux := http.NewServeMux()
	
	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "healthy",
			"active_conns":   atomic.LoadInt64(&mb.activeConns),
			"total_requests": atomic.LoadInt64(&mb.requestCount),
			"total_errors":   atomic.LoadInt64(&mb.errorCount),
		})
	})
	
	// Configuration endpoint
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mb.config)
	})

	httpPort := mb.config.Port + 1000 // HTTP on port+1000
	log.Printf("Starting HTTP server on port %d", httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), mux))
}

func (mb *MockBackend) startGRPCServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", mb.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(mb.unaryInterceptor),
	)

	// Register OTLP trace service
	ptraceotlp.RegisterGRPCServer(s, mb)

	log.Printf("Starting gRPC server on port %d", mb.config.Port)
	return s.Serve(lis)
}

func (mb *MockBackend) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	
	// Check concurrency limit
	currentConns := atomic.AddInt64(&mb.activeConns, 1)
	mb.activeConnGauge.Set(float64(currentConns))
	defer func() {
		newConns := atomic.AddInt64(&mb.activeConns, -1)
		mb.activeConnGauge.Set(float64(newConns))
	}()

	if currentConns > int64(mb.config.MaxConcurrency) {
		mb.errorsTotal.Inc()
		atomic.AddInt64(&mb.errorCount, 1)
		return nil, status.Error(codes.ResourceExhausted, "too many concurrent connections")
	}

	// Check backpressure threshold
	totalRequests := atomic.AddInt64(&mb.requestCount, 1)
	mb.requestsTotal.Inc()
	
	if totalRequests > mb.config.BackpressureThresh {
		// Simulate backpressure by returning errors for some requests
		if rand.Float64() > 0.5 { // 50% chance of error when over threshold
			mb.errorsTotal.Inc()
			atomic.AddInt64(&mb.errorCount, 1)
			return nil, status.Error(codes.Unavailable, "backend overloaded")
		}
	}

	// Simulate processing delay
	if mb.config.Delay > 0 {
		time.Sleep(mb.config.Delay)
	}

	// Simulate success rate
	if rand.Float64()*100 > mb.config.SuccessRate {
		mb.errorsTotal.Inc()
		atomic.AddInt64(&mb.errorCount, 1)
		return nil, status.Error(codes.Internal, "simulated backend error")
	}

	// Process the request
	resp, err := handler(ctx, req)
	
	// Record metrics
	duration := time.Since(start)
	mb.requestLatency.Observe(duration.Seconds())
	
	// Estimate request size (rough)
	if req != nil {
		mb.bytesReceived.Add(1024) // Rough estimate
		atomic.AddInt64(&mb.totalBytes, 1024)
	}

	return resp, err
}

// Implement ptraceotlp.GRPCServer interface
func (mb *MockBackend) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	// Count spans received
	spans := req.Traces().SpanCount()
	log.Printf("Received %d spans", spans)
	
	// Return success response
	return ptraceotlp.NewExportResponse(), nil
}

// Additional utility functions for testing

func (mb *MockBackend) UpdateConfig(newConfig *Config) {
	mb.config = newConfig
	log.Printf("Updated backend configuration: %+v", newConfig)
}

func (mb *MockBackend) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests": atomic.LoadInt64(&mb.requestCount),
		"active_conns":   atomic.LoadInt64(&mb.activeConns),
		"total_bytes":    atomic.LoadInt64(&mb.totalBytes),
		"total_errors":   atomic.LoadInt64(&mb.errorCount),
		"config":         mb.config,
	}
}

func (mb *MockBackend) Reset() {
	atomic.StoreInt64(&mb.requestCount, 0)
	atomic.StoreInt64(&mb.totalBytes, 0)
	atomic.StoreInt64(&mb.errorCount, 0)
	log.Println("Backend stats reset")
}