package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Best practice: use module/package name as tracer name
var tracer = otel.Tracer("github.com/odigos/stress-test/go-span-gen")

func getenvInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func generateSpan(ctx context.Context, payload string, spanBytes int) {
	// Use the passed context instead of context.Background()
	ctx, span := tracer.Start(ctx, "load-span")
	defer span.End() 

	// More comprehensive attributes following semantic conventions
	span.SetAttributes(
		attribute.String("payload", payload),
		attribute.String("lang", "go"),
		attribute.String("gen", "go-span-gen"),
		attribute.Int("payload_size", spanBytes),
		attribute.String("operation.type", "load-test"),
		attribute.Int("user.id", rand.Intn(10000)),
		attribute.String("request.id", strconv.FormatInt(time.Now().UnixNano(), 36)),
		attribute.Bool("trace.sampled", true),
		attribute.String("service.version", "1.0.0"),
		attribute.String("deployment.environment", "stress-test"),
	)
}

func main() {
	spansPerSec := getenvInt("SPANS_PER_SEC", 7000)
	spanBytes := getenvInt("SPAN_BYTES", 2000)
	payload := strings.Repeat("x", spanBytes)

	log.Printf("Starting Go span generator with %d spans/sec, %d bytes per span", spansPerSec, spanBytes)
	log.Printf("OTEL_SERVICE_NAME: %s", os.Getenv("OTEL_SERVICE_NAME"))
	log.Printf("OTEL_RESOURCE_ATTRIBUTES: %s", os.Getenv("OTEL_RESOURCE_ATTRIBUTES"))

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	spanCount := 0
	for range ticker.C {
		for i := 0; i < spansPerSec; i++ {
			// Use context.Background() only at the root level
			generateSpan(context.Background(), payload, spanBytes)
			spanCount++
		}
		log.Printf("Generated %d spans in this second (total: %d)", spansPerSec, spanCount)
	}
}
