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
	spansPerSec := getenvInt("SPANS_PER_SEC", 50)
	spanBytes := getenvInt("SPAN_BYTES", 10000)
	payload := strings.Repeat("x", spanBytes)

	log.Printf("Starting Go span generator with %d spans/sec, %d bytes per span", spansPerSec, spanBytes)
	log.Printf("OTEL_SERVICE_NAME: %s", os.Getenv("OTEL_SERVICE_NAME"))
	log.Printf("OTEL_RESOURCE_ATTRIBUTES: %s", os.Getenv("OTEL_RESOURCE_ATTRIBUTES"))

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	spanCount := 0
	for range ticker.C {
		for i := 0; i < spansPerSec; i++ {
			ctx, span := tracer.Start(context.Background(), "load-span")
			span.SetAttributes(
				attribute.String("payload", payload),
				attribute.String("lang", "go"),
				attribute.String("gen", "go-span-gen"),
			)
			span.End()
			_ = ctx
			spanCount++
		}
		log.Printf("Generated %d spans in this second (total: %d)", spansPerSec, spanCount)
	}
}
