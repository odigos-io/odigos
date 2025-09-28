package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Best practice: use module/package name as tracer name
var tracer = otel.Tracer("go-span-gen")

func getenvInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(name string, def float64) float64 {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

func main() {
	// Get configuration from environment variables
	spansPerSec := getenvInt("SPANS_PER_SEC", 1000)
	spanBytes := getenvInt("SPAN_BYTES", 1000)

	log.Printf("Starting Go span generator...")
	log.Printf("Configuration: %d spans/second, %d bytes per span", spansPerSec, spanBytes)

	// Create payload for attributes
	payload := strings.Repeat("x", spanBytes)

	// Create ticker for the specified rate
	intervalMs := 1000 / spansPerSec
	if intervalMs < 1 {
		intervalMs = 1 // Minimum 1ms interval
	}
	interval := time.Duration(intervalMs) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Generate spans continuously
	iteration := 0
	batchCount := 0
	for range ticker.C {
		// Create a new context for each span to ensure individual traces
		ctx := context.Background()

		_, span := tracer.Start(ctx, "go-span", trace.WithNewRoot())

		// Set attributes
		span.SetAttributes(
			attribute.String("payload", payload),
		)

		span.End()

		iteration++

		// Check if we've completed a full batch (1000 spans)
		if iteration%1000 == 0 {
			batchCount++
			log.Printf("Completed batch %d: Generated %d spans", batchCount, iteration)
		}
	}
}
