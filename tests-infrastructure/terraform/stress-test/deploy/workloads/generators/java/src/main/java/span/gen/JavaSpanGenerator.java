package span.gen;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.common.AttributeKey;

public class JavaSpanGenerator {
    public static void main(String[] args) {
        // Get configuration from environment variables
        int spansPerMinute = getEnvInt("SPANS_PER_MINUTE", 60);
        int attributeSize = getEnvInt("ATTRIBUTE_SIZE", 100);
        
        // Calculate delay between spans (in milliseconds)
        long delayMs = 60000 / spansPerMinute; // 60 seconds / spans per minute
        
        // Get tracer using GlobalOpenTelemetry
        Tracer tracer = GlobalOpenTelemetry.getTracer("java-span-gen", "1.0.0");
        
        System.out.println("Starting Java span generator...");
        System.out.println("Configuration: " + spansPerMinute + " spans/minute, " + attributeSize + " bytes per attribute");
        
        // Create payload for attributes
        String payload = "x".repeat(attributeSize);
        
        // Generate spans continuously
        int iteration = 0;
        while (true) {
            Span span = tracer.spanBuilder("configurable-span")
                .setAttribute(AttributeKey.stringKey("lang"), "java")
                .setAttribute(AttributeKey.stringKey("iteration"), String.valueOf(iteration))
                .setAttribute(AttributeKey.stringKey("payload"), payload)
                .setAttribute(AttributeKey.longKey("attribute_size"), (long) attributeSize)
                .setAttribute(AttributeKey.longKey("spans_per_minute"), (long) spansPerMinute)
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
            if (iteration % 10 == 0) {
                System.out.println("Generated " + iteration + " spans so far...");
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
