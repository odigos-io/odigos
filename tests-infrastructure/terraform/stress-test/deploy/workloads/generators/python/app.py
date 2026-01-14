import os
import time
import logging
import threading
from concurrent.futures import ThreadPoolExecutor
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

def emit_batch_optimized(spans_per_sec: int, span_bytes: int, payload: str, batch_size: int = 100):
    """Generate a batch of spans with optimizations."""
    try:
        # Pre-create attributes to avoid repeated string operations
        attributes = {
            "payload": payload,
            "lang": "python",
            "gen": "py-span-gen",
            "payload_size": span_bytes
        }
        
        # Process spans in batches to reduce overhead
        for i in range(0, spans_per_sec, batch_size):
            current_batch_size = min(batch_size, spans_per_sec - i)
            
            # Use context manager more efficiently
            with tracer.start_as_current_span("load-span-batch") as batch_span:
                batch_span.set_attributes(attributes)
                
                # Generate individual spans within the batch
                for _ in range(current_batch_size):
                    with tracer.start_as_current_span("load-span") as span:
                        span.set_attributes(attributes)
                        
    except Exception as e:
        logger.error("Error generating spans: %s", e)

def emit_batch_parallel(spans_per_sec: int, span_bytes: int, payload: str, max_workers: int = 4):
    """Generate spans using parallel processing."""
    try:
        # Pre-create attributes
        attributes = {
            "payload": payload,
            "lang": "python", 
            "gen": "py-span-gen",
            "payload_size": span_bytes
        }
        
        def generate_span_batch(batch_size):
            for _ in range(batch_size):
                with tracer.start_as_current_span("load-span") as span:
                    span.set_attributes(attributes)
        
        # Distribute work across threads
        spans_per_thread = spans_per_sec // max_workers
        remaining_spans = spans_per_sec % max_workers
        
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = []
            
            # Submit work to threads
            for i in range(max_workers):
                batch_size = spans_per_thread + (1 if i < remaining_spans else 0)
                if batch_size > 0:
                    futures.append(executor.submit(generate_span_batch, batch_size))
            
            # Wait for all threads to complete
            for future in futures:
                future.result()
                
    except Exception as e:
        logger.error("Error generating spans in parallel: %s", e)

def main():
    # Get configuration from environment
    spans_per_sec = getenv_int("SPANS_PER_SEC", 1000)
    span_bytes = getenv_int("SPAN_BYTES", 500)
    use_parallel = os.getenv("USE_PARALLEL", "true").lower() == "true"
    max_workers = getenv_int("MAX_WORKERS", 4)
    batch_size = getenv_int("BATCH_SIZE", 100)
    
    payload = "x" * span_bytes
    
    # Log startup information
    logger.info("Starting Python span generator with %d spans/sec, %d bytes per span", 
                spans_per_sec, span_bytes)
    logger.info("Parallel processing: %s, Max workers: %d, Batch size: %d", 
                use_parallel, max_workers, batch_size)
    
    span_count = 0
    performance_stats = {"total_time": 0, "generation_time": 0, "sleep_time": 0}
    
    while True:
        start_time = time.time()
        
        # Generate spans using optimized method
        generation_start = time.time()
        if use_parallel:
            emit_batch_parallel(spans_per_sec, span_bytes, payload, max_workers)
        else:
            emit_batch_optimized(spans_per_sec, span_bytes, payload, batch_size)
        generation_time = time.time() - generation_start
        
        span_count += spans_per_sec
        
        # Calculate elapsed time and sleep if needed
        elapsed = time.time() - start_time
        sleep_time = max(0.0, 1.0 - elapsed)
        
        if sleep_time > 0:
            time.sleep(sleep_time)
        
        # Update performance stats
        performance_stats["total_time"] += time.time() - start_time
        performance_stats["generation_time"] += generation_time
        performance_stats["sleep_time"] += sleep_time
        
        # Log progress with performance metrics
        cpu_efficiency = (generation_time / (generation_time + sleep_time)) * 100 if (generation_time + sleep_time) > 0 else 0
        logger.info("Generated %d spans in %.3fs (gen: %.3fs, sleep: %.3fs, CPU efficiency: %.1f%%)", 
                   spans_per_sec, elapsed, generation_time, sleep_time, cpu_efficiency)
        
        # Log performance summary every 60 seconds
        if span_count % (spans_per_sec * 60) == 0:
            avg_total = performance_stats["total_time"] / 60
            avg_gen = performance_stats["generation_time"] / 60
            avg_sleep = performance_stats["sleep_time"] / 60
            logger.info("Performance summary (last 60s): avg total=%.3fs, gen=%.3fs, sleep=%.3fs", 
                       avg_total, avg_gen, avg_sleep)
            performance_stats = {"total_time": 0, "generation_time": 0, "sleep_time": 0}

if __name__ == "__main__":
    main()
