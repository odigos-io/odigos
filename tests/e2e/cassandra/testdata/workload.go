package main

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"time"
)

func main() {
	ctx := context.Background()
	for _, name := range []string{"checkout-service", "inventory-service"} {
		res := resource.NewSchemaless(attribute.String("service.name", name), attribute.String("deployment.environment.name", "core-1772-e2e"))
		tx, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint("localhost:14318"), otlptracehttp.WithInsecure())
		if err != nil {
			panic(err)
		}
		tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(tx))
		lx, err := otlploghttp.New(ctx, otlploghttp.WithEndpoint("localhost:14318"), otlploghttp.WithInsecure())
		if err != nil {
			panic(err)
		}
		lp := sdklog.NewLoggerProvider(sdklog.WithResource(res), sdklog.WithProcessor(sdklog.NewBatchProcessor(lx)))
		for i := 0; i < 3; i++ {
			spanctx, span := tp.Tracer("ordinary-http-workload").Start(ctx, "GET /orders")
			span.SetAttributes(attribute.Int("http.response.status_code", 200), attribute.String("http.request.method", "GET"))
			var record otellog.Record
			record.SetTimestamp(time.Now())
			record.SetSeverity(otellog.SeverityInfo)
			record.SetSeverityText("INFO")
			record.SetBody(attribute.StringValue(fmt.Sprintf("completed request %d", i)))
			record.AddAttributes(attribute.String("request.path", "/orders"), attribute.Int("http.status_code", 200))
			lp.Logger("application").Emit(spanctx, record)
			span.End()
		}
		// Ordinary application logs often have no current span.
		var record otellog.Record
		record.SetTimestamp(time.Now())
		record.SetSeverity(otellog.SeverityInfo)
		record.SetBody(attribute.MapValue(attribute.String("message", "background job finished"), attribute.Bool("success", true)))
		lp.Logger("background").Emit(ctx, record)
		if err := tp.Shutdown(ctx); err != nil {
			panic(err)
		}
		if err := lp.Shutdown(ctx); err != nil {
			panic(err)
		}
	}
	fmt.Println("exported application traces and logs from two services through OTLP HTTP")
}
