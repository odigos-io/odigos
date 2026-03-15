package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var urlPatterns = []struct {
	pattern   string
	generator func() string
}{
	{pattern: "/api/users/{id}", generator: func() string { return fmt.Sprintf("/api/users/%s", randomID("u")) }},
	{pattern: "/api/items/{itemId}", generator: func() string { return fmt.Sprintf("/api/items/%s", randomID("item")) }},
	{pattern: "/api/users/{userId}/orders/{orderId}", generator: func() string { return fmt.Sprintf("/api/users/%d/orders/%s", rand.Intn(100000), randomID("ord")) }},
	{pattern: "/v1/tenants/{tenantId}/resources/{resourceId}", generator: func() string { return fmt.Sprintf("/v1/tenants/%s/resources/%s", randomUUID(), randomUUID()) }},
}

func randomUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(), rand.Uint32()&0xFFFF, rand.Uint32()&0xFFFF,
		rand.Uint32()&0xFFFF, rand.Uint64()&0xFFFFFFFFFFFF)
}

func randomID(prefix string) string { return fmt.Sprintf("%s-%d", prefix, rand.Intn(1000000)) }

var tracer trace.Tracer

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "url-demo-app"
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("deployment.environment", "demo"),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	tracer = tp.Tracer("url-demo-app")
	return tp, nil
}

func main() {
	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "path": "%s"}`, r.URL.Path)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	handler := otelhttp.NewHandler(mux, "url-demo-app",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
	go generateTraffic(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting URL Demo App on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func generateTraffic(ctx context.Context) {
	time.Sleep(3 * time.Second)
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   5 * time.Second,
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			pattern := urlPatterns[rand.Intn(len(urlPatterns))]
			url := baseURL + pattern.generator()
			method := methods[rand.Intn(len(methods))]
			ctx, parentSpan := tracer.Start(ctx, "client-request", trace.WithAttributes(attribute.String("url.pattern", pattern.pattern)))
			req, err := http.NewRequestWithContext(ctx, method, url, nil)
			if err != nil {
				parentSpan.End()
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				parentSpan.End()
				time.Sleep(500 * time.Millisecond)
				continue
			}
			resp.Body.Close()
			parentSpan.End()
			time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)
		}
	}
}
