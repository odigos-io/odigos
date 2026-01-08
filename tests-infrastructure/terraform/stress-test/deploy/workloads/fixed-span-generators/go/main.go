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

var tracer = otel.Tracer("github.com/odigos/fixed-span-gen/go")

func getenvInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func generateSpan(ctx context.Context, payload string, spanBytes int, spanNum int) {
	// Span start time is set here
	ctx, span := tracer.Start(ctx, "fixed-span")
	// Span end time is set here
	defer span.End()

	// ...Do some work, use ctx if needed when calling next functions
	span.SetAttributes(
		attribute.String("payload", payload),
		attribute.String("lang", "go"),
		attribute.String("gen", "go-fixed-span-gen"),
		attribute.Int("payload_size", spanBytes),
		attribute.Int("span_number", spanNum),
		attribute.String("operation.type", "fixed-load-test"),
		attribute.Int("user.id", rand.Intn(10000)),
		attribute.String("request.id", strconv.FormatInt(time.Now().UnixNano(), 36)),
		attribute.Bool("trace.sampled", true),
		attribute.String("service.version", "1.0.0"),
		attribute.String("deployment.environment", "fixed-span-test"),
	)
}

func main() {
	totalSpans := getenvInt("TOTAL_SPANS", 10000)
	spanBytes := getenvInt("SPAN_BYTES", 2000)
	payload := strings.Repeat("x", spanBytes)

	log.Printf("Starting Go fixed span generator with %d total spans, %d bytes per span", totalSpans, spanBytes)

	startTime := time.Now()
	spanCount := 0

	// Generate all spans in a single batch
	for i := 0; i < totalSpans; i++ {
		// Use context.Background() only at the root level
		generateSpan(context.Background(), payload, spanBytes, i+1)
		spanCount++

		// Log progress every 1000 spans
		if spanCount%1000 == 0 {
			log.Printf("Generated %d/%d spans (%.1f%%)", spanCount, totalSpans, float64(spanCount)/float64(totalSpans)*100)
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("Completed generating %d spans in %.2f seconds (%.2f spans/sec)",
		spanCount, elapsed.Seconds(), float64(spanCount)/elapsed.Seconds())

	// Keep the container running for a bit to ensure all spans are exported
	log.Printf("Waiting 30 seconds to ensure all spans are exported...")
	time.Sleep(30 * time.Second)

	log.Printf("Go fixed span generator completed successfully!")
}
