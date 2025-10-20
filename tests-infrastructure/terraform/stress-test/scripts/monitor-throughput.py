#!/usr/bin/env python3
"""
Stress test throughput monitor for OpenTelemetry Collector metrics.

This script monitors the otelcol_exporter_sent_spans__spans__total metric
and calculates throughput statistics while running span generators.
"""

import os
import time
import json
import logging
import argparse
import requests
import statistics
from datetime import datetime, timedelta
from typing import List, Dict, Optional
from dataclasses import dataclass, asdict

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s %(levelname)s %(message)s',
    datefmt='%Y/%m/%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

@dataclass
class MetricSample:
    """Represents a single metric sample."""
    timestamp: float
    value: float
    labels: Dict[str, str]

@dataclass
class ThroughputStats:
    """Throughput statistics for a time period."""
    start_time: float
    end_time: float
    duration_seconds: float
    total_spans: int
    avg_throughput_spans_per_sec: float
    min_throughput_spans_per_sec: float
    max_throughput_spans_per_sec: float
    samples_count: int

class MetricsCollector:
    """Collects and processes OpenTelemetry Collector metrics."""
    
    def __init__(self, metrics_url: str = "http://localhost:8888/metrics"):
        self.metrics_url = metrics_url
        self.samples: List[MetricSample] = []
        
    def fetch_metrics(self) -> Optional[MetricSample]:
        """Fetch metrics from the collector and parse the spans counter."""
        try:
            response = requests.get(self.metrics_url, timeout=5)
            response.raise_for_status()
            
            current_time = time.time()
            spans_value = self._parse_spans_metric(response.text)
            
            if spans_value is not None:
                sample = MetricSample(
                    timestamp=current_time,
                    value=spans_value,
                    labels={}
                )
                self.samples.append(sample)
                return sample
                
        except requests.RequestException as e:
            logger.error("Failed to fetch metrics: %s", e)
        except Exception as e:
            logger.error("Error parsing metrics: %s", e)
            
        return None
    
    def _parse_spans_metric(self, metrics_text: str) -> Optional[float]:
        """Parse the otelcol_exporter_sent_spans__spans__total metric from Prometheus format."""
        for line in metrics_text.split('\n'):
            line = line.strip()
            if line.startswith('otelcol_exporter_sent_spans__spans__total'):
                # Handle both labeled and unlabeled metrics
                if '{' in line:
                    # Labeled metric: otelcol_exporter_sent_spans__spans__total{exporter="otlp",pipeline="traces/otlp"} 12345
                    parts = line.split(' ')
                    if len(parts) >= 2:
                        try:
                            return float(parts[-1])
                        except ValueError:
                            continue
                else:
                    # Unlabeled metric: otelcol_exporter_sent_spans__spans__total 12345
                    parts = line.split(' ')
                    if len(parts) >= 2:
                        try:
                            return float(parts[1])
                        except ValueError:
                            continue
        return None
    
    def calculate_throughput_stats(self, start_time: float, end_time: float) -> Optional[ThroughputStats]:
        """Calculate throughput statistics for a given time period."""
        # Filter samples within the time range
        period_samples = [
            s for s in self.samples 
            if start_time <= s.timestamp <= end_time
        ]
        
        if len(period_samples) < 2:
            return None
            
        # Sort by timestamp
        period_samples.sort(key=lambda x: x.timestamp)
        
        # Calculate throughput for each interval
        throughputs = []
        for i in range(1, len(period_samples)):
            prev_sample = period_samples[i-1]
            curr_sample = period_samples[i]
            
            time_delta = curr_sample.timestamp - prev_sample.timestamp
            value_delta = curr_sample.value - prev_sample.value
            
            if time_delta > 0:
                throughput = value_delta / time_delta
                throughputs.append(throughput)
        
        if not throughputs:
            return None
            
        # Calculate total spans sent during the period
        total_spans = period_samples[-1].value - period_samples[0].value
        
        return ThroughputStats(
            start_time=start_time,
            end_time=end_time,
            duration_seconds=end_time - start_time,
            total_spans=int(total_spans),
            avg_throughput_spans_per_sec=statistics.mean(throughputs),
            min_throughput_spans_per_sec=min(throughputs),
            max_throughput_spans_per_sec=max(throughputs),
            samples_count=len(throughputs)
        )

class StressTestMonitor:
    """Main stress test monitoring class."""
    
    def __init__(self, metrics_url: str, interval_seconds: int = 5, span_bytes: int = 0):
        self.collector = MetricsCollector(metrics_url)
        self.interval_seconds = interval_seconds
        self.start_time = None
        self.test_duration = None
        self.results = []
        self.span_bytes = span_bytes  # 0 means unknown
        
    def run_test(self, duration_minutes: int, output_file: Optional[str] = None):
        """Run the stress test for the specified duration."""
        self.start_time = time.time()
        self.test_duration = duration_minutes * 60
        
        logger.info("Starting stress test monitor for %d minutes", duration_minutes)
        logger.info("Monitoring metrics at: %s", self.collector.metrics_url)
        logger.info("Sampling interval: %d seconds", self.interval_seconds)
        
        end_time = self.start_time + self.test_duration
        last_sample_time = None
        
        try:
            while time.time() < end_time:
                current_time = time.time()
                
                # Fetch metrics
                sample = self.collector.fetch_metrics()
                if sample:
                    if last_sample_time:
                        # Calculate instantaneous throughput
                        time_delta = sample.timestamp - last_sample_time
                        value_delta = sample.value - last_sample_time
                        if time_delta > 0:
                            instant_throughput = value_delta / time_delta
                            if self.span_bytes and self.span_bytes > 0:
                                mbps = (instant_throughput * self.span_bytes) / 1_000_000.0
                                logger.info(
                                    "Current throughput: %.2f spans/sec, %.3f MB/sec (total: %.0f)",
                                    instant_throughput, mbps, sample.value,
                                )
                            else:
                                logger.info(
                                    "Current throughput: %.2f spans/sec (total: %.0f)",
                                    instant_throughput, sample.value,
                                )
                    
                    last_sample_time = sample.timestamp
                else:
                    logger.warning("Failed to fetch metrics at %s", 
                                 datetime.fromtimestamp(current_time).strftime('%H:%M:%S'))
                
                # Sleep until next interval
                sleep_time = self.interval_seconds - (time.time() - current_time)
                if sleep_time > 0:
                    time.sleep(sleep_time)
                    
        except KeyboardInterrupt:
            logger.info("Test interrupted by user")
        
        # Calculate final statistics
        self._calculate_final_stats()
        
        # Save results if requested
        if output_file:
            self._save_results(output_file)
            
        self._print_summary()
    
    def _calculate_final_stats(self):
        """Calculate final statistics for the entire test."""
        if not self.collector.samples:
            logger.error("No samples collected during test")
            return
            
        # Calculate overall statistics
        overall_stats = self.collector.calculate_throughput_stats(
            self.start_time, 
            self.start_time + self.test_duration
        )
        
        if overall_stats:
            self.results.append(overall_stats)
            
        # Calculate statistics for each minute
        for minute in range(int(self.test_duration / 60)):
            minute_start = self.start_time + (minute * 60)
            minute_end = minute_start + 60
            
            minute_stats = self.collector.calculate_throughput_stats(minute_start, minute_end)
            if minute_stats:
                self.results.append(minute_stats)
    
    def _save_results(self, output_file: str):
        """Save results to JSON file."""
        results_data = {
            "test_info": {
                "start_time": datetime.fromtimestamp(self.start_time).isoformat(),
                "duration_minutes": self.test_duration / 60,
                "interval_seconds": self.interval_seconds,
                "metrics_url": self.collector.metrics_url
            },
            "results": [asdict(result) for result in self.results]
        }
        
        with open(output_file, 'w') as f:
            json.dump(results_data, f, indent=2)
            
        logger.info("Results saved to: %s", output_file)
    
    def _print_summary(self):
        """Print test summary."""
        if not self.results:
            logger.error("No results to summarize")
            return
            
        logger.info("=" * 60)
        logger.info("STRESS TEST SUMMARY")
        logger.info("=" * 60)
        
        # Overall statistics
        overall = self.results[0]  # First result is overall
        logger.info("Overall Test Results:")
        logger.info("  Duration: %.1f minutes", overall.duration_seconds / 60)
        logger.info("  Total Spans Sent: %d", overall.total_spans)
        logger.info("  Average Throughput: %.2f spans/sec", overall.avg_throughput_spans_per_sec)
        if self.span_bytes and self.span_bytes > 0:
            avg_mbps = (overall.avg_throughput_spans_per_sec * self.span_bytes) / 1_000_000.0
            logger.info("  Average Payload Throughput: %.3f MB/sec (payload only)", avg_mbps)
        logger.info("  Min Throughput: %.2f spans/sec", overall.min_throughput_spans_per_sec)
        logger.info("  Max Throughput: %.2f spans/sec", overall.max_throughput_spans_per_sec)
        logger.info("  Samples Collected: %d", overall.samples_count)
        
        # Per-minute statistics
        if len(self.results) > 1:
            logger.info("\nPer-Minute Breakdown:")
            minute_results = self.results[1:]  # Skip overall result
            
            throughputs = [r.avg_throughput_spans_per_sec for r in minute_results]
            if throughputs:
                logger.info("  Average per-minute throughput: %.2f spans/sec", statistics.mean(throughputs))
                if self.span_bytes and self.span_bytes > 0:
                    logger.info(
                        "  Average per-minute payload throughput: %.3f MB/sec",
                        (statistics.mean(throughputs) * self.span_bytes) / 1_000_000.0,
                    )
                logger.info("  Min per-minute throughput: %.2f spans/sec", min(throughputs))
                logger.info("  Max per-minute throughput: %.2f spans/sec", max(throughputs))
                logger.info("  Throughput std deviation: %.2f spans/sec", statistics.stdev(throughputs) if len(throughputs) > 1 else 0)

def main():
    parser = argparse.ArgumentParser(description="Monitor OpenTelemetry Collector throughput during stress tests")
    parser.add_argument("--metrics-url", default="http://localhost:8888/metrics",
                       help="URL to fetch metrics from (default: http://localhost:8888/metrics)")
    parser.add_argument("--duration", type=int, default=5,
                       help="Test duration in minutes (default: 5)")
    parser.add_argument("--interval", type=int, default=5,
                       help="Sampling interval in seconds (default: 5)")
    parser.add_argument("--span-bytes", type=int, default=0,
                       help="Known per-span payload size in bytes (optional). If set, MB/sec is computed")
    parser.add_argument("--output", help="Output file for results (JSON format)")
    
    args = parser.parse_args()
    
    monitor = StressTestMonitor(args.metrics_url, args.interval, args.span_bytes)
    monitor.run_test(args.duration, args.output)

if __name__ == "__main__":
    main()
