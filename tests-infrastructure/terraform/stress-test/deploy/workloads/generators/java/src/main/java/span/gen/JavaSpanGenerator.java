package span.gen;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.common.AttributeKey;

public class JavaSpanGenerator {
    public static void main(String[] args) {
        // Get configuration from environment variables
        int spansPerSec = getEnvInt("SPANS_PER_SEC", 1000);
        int spanBytes = getEnvInt("SPAN_BYTES", 1000);
        
        // Calculate delay between spans (in milliseconds)
        long delayMs = 1000 / spansPerSec; // 1000ms / spans per second
        
        // Get tracer using GlobalOpenTelemetry
        Tracer tracer = GlobalOpenTelemetry.getTracer("java-span-gen", "1.0.0");
        
        System.out.println("Starting Java span generator...");
        System.out.println("Configuration: " + spansPerSec + " spans/second, " + spanBytes + " bytes per span");
        
        // Create payload for attributes
        String payload = "x".repeat(spanBytes);
        
        // Generate spans continuously
        int iteration = 0;
        while (true) {
            Span span = tracer.spanBuilder("java-span-" + iteration)
                .setAttribute(AttributeKey.stringKey("payload"), payload)
                .startSpan();
            
            try {
                // Wait for the calculated delay
                Thread.sleep(delayMs);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            } finally {
                span.end();
            }
            
            iteration++;
            if (iteration % 1000 == 0) {
                System.out.println("Completed batch: Generated " + iteration + " spans");
            }
        }
        
        System.out.println("Java span generator stopped.");
    }
    
    private static int getEnvInt(String name, int defaultValue) {
        try {
            String value = System.getenv(name);
            if (value != null && !value.trim().isEmpty()) {
                return Integer.parseInt(value.trim());
            }
        } catch (NumberFormatException e) {
            System.err.println("Invalid value for " + name + ", using default: " + defaultValue);
        }
        return defaultValue;
    }
}
