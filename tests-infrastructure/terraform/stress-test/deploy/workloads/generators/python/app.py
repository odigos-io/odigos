import os
import time
import logging
from opentelemetry import trace

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s %(levelname)s %(message)s',
    datefmt='%Y/%m/%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

# Get tracer
tracer = trace.get_tracer(__name__)

def getenv_int(name: str, default: int) -> int:
    """Get environment variable as integer with default value."""
    try:
        return int(os.getenv(name, str(default)))
    except ValueError:
        return default

def emit_batch(spans_per_sec: int, span_bytes: int, payload: str):
    """Generate a batch of spans."""
    try:
        for _ in range(spans_per_sec):
            with tracer.start_as_current_span("load-span") as span:
                span.set_attribute("payload", payload)
                span.set_attribute("lang", "python")
                span.set_attribute("gen", "py-span-gen")
                span.set_attribute("payload_size", span_bytes)
    except Exception as e:
        logger.error("Error generating spans: %s", e)

def main():
    # Get configuration from environment
    spans_per_sec = getenv_int("SPANS_PER_SEC", 1000)
    span_bytes = getenv_int("SPAN_BYTES", 500)
    payload = "x" * span_bytes
    
    # Log startup information
    logger.info("Starting Python span generator with %d spans/sec, %d bytes per span", 
                spans_per_sec, span_bytes)
    logger.info("OTEL_SERVICE_NAME: %s", os.getenv("OTEL_SERVICE_NAME", "Not set"))
    logger.info("OTEL_RESOURCE_ATTRIBUTES: %s", 
                os.getenv("OTEL_RESOURCE_ATTRIBUTES", "Not set"))
    
    span_count = 0
    
    while True:
        start_time = time.time()
        
        # Generate spans
        emit_batch(spans_per_sec, span_bytes, payload)
        span_count += spans_per_sec
        
        # Calculate elapsed time and sleep if needed
        elapsed = time.time() - start_time
        sleep_time = max(0.0, 1.0 - elapsed)
        
        if sleep_time > 0:
            time.sleep(sleep_time)
        
        # Log progress
        logger.info("Generated %d spans in this second (total: %d)", spans_per_sec, span_count)

if __name__ == "__main__":
    main()
