package span.gen;

import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.trace.TracerProvider;
import io.opentelemetry.api.common.AttributeKey;
import io.opentelemetry.api.common.Attributes;

import java.util.logging.Logger;
import java.util.logging.Level;

public class JavaFixedSpanGenerator {
    private static final Logger logger = Logger.getLogger(JavaFixedSpanGenerator.class.getName());
    
    public static void main(String[] args) {
        // Get configuration from environment variables
        int totalSpans = getEnvInt("TOTAL_SPANS", 10000);
        int spanBytes = getEnvInt("SPAN_BYTES", 2000);
        
        // Log startup information
        logger.info("Starting Java fixed span generator with " + totalSpans + " total spans, " + spanBytes + " bytes per span");
        logger.info("OTEL_SERVICE_NAME: " + System.getenv("OTEL_SERVICE_NAME"));
        logger.info("OTEL_RESOURCE_ATTRIBUTES: " + System.getenv("OTEL_RESOURCE_ATTRIBUTES"));
        
        // Create payload
        String payload = "x".repeat(spanBytes);
        
        // Get tracer
        TracerProvider tracerProvider = TracerProvider.noop();
        Tracer tracer = tracerProvider.get("java-fixed-span-gen");
        
        long startTime = System.currentTimeMillis();
        int generatedSpans = 0;
        
        logger.info("Starting to generate " + totalSpans + " spans...");
        
        try {
            // Generate all spans
            for (int i = 0; i < totalSpans; i++) {
                Span span = tracer.spanBuilder("fixed-span")
                    .setAttribute(AttributeKey.stringKey("payload"), payload)
                    .setAttribute(AttributeKey.stringKey("lang"), "java")
                    .setAttribute(AttributeKey.stringKey("gen"), "java-fixed-span-gen")
                    .setAttribute(AttributeKey.longKey("payload_size"), (long) spanBytes)
                    .setAttribute(AttributeKey.longKey("span_number"), (long) (i + 1))
                    .setAttribute(AttributeKey.stringKey("operation.type"), "fixed-load-test")
                    .setAttribute(AttributeKey.longKey("user.id"), (long) ((i + 1) % 10000))
                    .setAttribute(AttributeKey.stringKey("request.id"), "req-" + (i + 1) + "-" + System.currentTimeMillis())
                    .setAttribute(AttributeKey.booleanKey("trace.sampled"), true)
                    .setAttribute(AttributeKey.stringKey("service.version"), "1.0.0")
                    .setAttribute(AttributeKey.stringKey("deployment.environment"), "fixed-span-test")
                    .startSpan();
                
                // Simulate some work
                try {
                    // Add small delay to reduce CPU usage
                    if (i % 100 == 0) {
                        Thread.sleep(1);
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } finally {
                    span.end();
                }
                
                generatedSpans++;
                
                // Log progress every 1000 spans
                if (generatedSpans % 1000 == 0) {
                    double progress = (double) generatedSpans / totalSpans * 100;
                    logger.info("Generated " + generatedSpans + "/" + totalSpans + " spans (" + String.format("%.1f", progress) + "%)");
                }
            }
            
            long elapsed = System.currentTimeMillis() - startTime;
            double elapsedSeconds = elapsed / 1000.0;
            double spansPerSec = totalSpans / elapsedSeconds;
            
            logger.info("Completed generating " + totalSpans + " spans in " + String.format("%.2f", elapsedSeconds) + 
                       " seconds (" + String.format("%.2f", spansPerSec) + " spans/sec)");
            
            // Keep the container running for a bit to ensure all spans are exported
            logger.info("Waiting 30 seconds to ensure all spans are exported...");
            Thread.sleep(30000);
            
            logger.info("Java fixed span generator completed successfully!");
            
        } catch (Exception e) {
            logger.log(Level.SEVERE, "Error generating spans", e);
        }
    }
    
    private static int getEnvInt(String name, int defaultValue) {
        try {
            String value = System.getenv(name);
            if (value != null && !value.trim().isEmpty()) {
                return Integer.parseInt(value.trim());
            }
        } catch (NumberFormatException e) {
            logger.warning("Invalid value for " + name + ", using default: " + defaultValue);
        }
        return defaultValue;
    }
}
