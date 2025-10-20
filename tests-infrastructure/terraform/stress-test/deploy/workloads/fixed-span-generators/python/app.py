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

def generate_span(span_num: int, payload: str, span_bytes: int):
    """Generate a single span with attributes."""
    with tracer.start_as_current_span("fixed-span") as span:
        span.set_attributes({
            "payload": payload,
            "lang": "python",
            "gen": "py-fixed-span-gen",
            "payload_size": span_bytes,
            "span_number": span_num,
            "operation.type": "fixed-load-test",
            "user.id": span_num % 10000,
            "request.id": f"req-{span_num}-{int(time.time() * 1000000)}",
            "trace.sampled": True,
            "service.version": "1.0.0",
            "deployment.environment": "fixed-span-test"
        })

def main():
    # Get configuration from environment
    total_spans = getenv_int("TOTAL_SPANS", 10000)
    span_bytes = getenv_int("SPAN_BYTES", 2000)
    
    payload = "x" * span_bytes
    
    # Log startup information
    logger.info("Starting Python fixed span generator with %d total spans, %d bytes per span", 
                total_spans, span_bytes)
    
    start_time = time.time()
    
    # Generate all spans
    for i in range(total_spans):
        generate_span(i + 1, payload, span_bytes)
        
        # Log progress every 1000 spans
        if (i + 1) % 1000 == 0:
            progress = (i + 1) / total_spans * 100
            logger.info("Generated %d/%d spans (%.1f%%)", i + 1, total_spans, progress)
    
    elapsed = time.time() - start_time
    logger.info("Completed generating %d spans in %.2f seconds (%.2f spans/sec)", 
                total_spans, elapsed, total_spans / elapsed)
    
    # Keep the container running for a bit to ensure all spans are exported
    logger.info("Waiting 30 seconds to ensure all spans are exported...")
    time.sleep(30)
    
    logger.info("Python fixed span generator completed successfully!")

if __name__ == "__main__":
    main()
