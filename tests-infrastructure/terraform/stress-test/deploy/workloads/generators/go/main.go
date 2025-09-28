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

func main() {
	// Get configuration from environment variables
	spansPerMinute := getenvInt("SPANS_PER_MINUTE", 60)
	attributeSize := getenvInt("ATTRIBUTE_SIZE", 100)

	log.Printf("Starting Go span generator...")
	log.Printf("Configuration: %d spans/minute, %d bytes per attribute", spansPerMinute, attributeSize)

	// Create payload for attributes
	payload := strings.Repeat("x", attributeSize)

	// Create ticker for the specified rate
	interval := time.Duration(60/spansPerMinute) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Generate spans continuously
	iteration := 0
	batchCount := 0
	for range ticker.C {
		// Create a new context for each span to ensure individual traces
		ctx := context.Background()

		// Create a new span with a unique name for each iteration
		spanName := "go-span-" + strconv.Itoa(iteration)
		_, span := tracer.Start(ctx, spanName, trace.WithNewRoot())

		// Set attributes
		span.SetAttributes(
			attribute.String("lang", "go"),
			attribute.String("iteration", strconv.Itoa(iteration)),
			attribute.String("payload", payload),
			attribute.Int("attribute_size", attributeSize),
			attribute.Int("spans_per_minute", spansPerMinute),
		)

		span.End()

		iteration++

		// Check if we've completed a full batch (minute's worth of spans)
		if iteration%spansPerMinute == 0 {
			batchCount++
			log.Printf("Completed batch %d: Generated %d spans", batchCount, iteration)
		}
	}
}
