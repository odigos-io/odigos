#!/usr/bin/env python3

import argparse
import json
import os
import sys
import csv
import subprocess
from datetime import datetime, timedelta
from pathlib import Path
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import pandas as pd
import numpy as np
from dataclasses import dataclass
from typing import List, Dict, Optional

@dataclass
class TestResult:
    test_id: str
    scenario: str
    start_time: str
    end_time: str
    duration: str
    spans_per_second: int
    cpu_limit: str
    memory_limit: str
    namespace: str

class ResultAnalyzer:
    def __init__(self, result_dir: str):
        self.result_dir = Path(result_dir)
        self.graphs_dir = self.result_dir / "graphs"
        self.graphs_dir.mkdir(exist_ok=True)
        
        # Load test summary
        summary_file = self.result_dir / "summary.json"
        if summary_file.exists():
            with open(summary_file) as f:
                summary_data = json.load(f)
                self.test_result = TestResult(**summary_data)
        else:
            print(f"Warning: summary.json not found in {result_dir}")
            self.test_result = None

    def analyze(self):
        """Run complete analysis and generate all reports"""
        print(f"Analyzing results in {self.result_dir}")
        
        # Generate resource usage graphs
        self.analyze_resource_usage()
        
        # Analyze collector metrics
        self.analyze_collector_metrics()
        
        # Generate summary report
        self.generate_summary_report()
        
        # Generate performance recommendations
        self.generate_recommendations()
        
        print(f"Analysis complete. Results saved to {self.result_dir}")

    def analyze_resource_usage(self):
        """Analyze CPU and memory usage over time"""
        resource_file = self.result_dir / "resource-usage.csv"
        if not resource_file.exists():
            print("Warning: resource-usage.csv not found")
            return

        try:
            # Read resource usage data
            df = pd.read_csv(resource_file)
            df['timestamp'] = pd.to_datetime(df['timestamp'], unit='s')
            df['cpu_percent'] = df['cpu'].str.replace('m', '').astype(float) / 10  # Convert millicores to percent
            df['memory_mb'] = df['memory'].str.replace('Mi', '').astype(float)

            # Create resource usage graphs
            fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(12, 10))
            
            # CPU usage
            ax1.plot(df['timestamp'], df['cpu_percent'], 'b-', linewidth=2, label='CPU Usage')
            ax1.axhline(y=80, color='r', linestyle='--', alpha=0.7, label='Warning Threshold (80%)')
            ax1.set_ylabel('CPU Usage (%)')
            ax1.set_title(f'CPU Usage Over Time - {self.test_result.test_id if self.test_result else "Unknown Test"}')
            ax1.legend()
            ax1.grid(True, alpha=0.3)
            ax1.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M:%S'))
            
            # Memory usage
            ax2.plot(df['timestamp'], df['memory_mb'], 'g-', linewidth=2, label='Memory Usage')
            if self.test_result:
                memory_limit = float(self.test_result.memory_limit.replace('Gi', '')) * 1024
                ax2.axhline(y=memory_limit, color='r', linestyle='--', alpha=0.7, label=f'Memory Limit ({memory_limit:.0f}MB)')
                ax2.axhline(y=memory_limit * 0.8, color='orange', linestyle='--', alpha=0.7, label='Warning Threshold (80%)')
            ax2.set_ylabel('Memory Usage (MB)')
            ax2.set_xlabel('Time')
            ax2.set_title('Memory Usage Over Time')
            ax2.legend()
            ax2.grid(True, alpha=0.3)
            ax2.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M:%S'))
            
            plt.xticks(rotation=45)
            plt.tight_layout()
            plt.savefig(self.graphs_dir / "resource-usage.png", dpi=300, bbox_inches='tight')
            plt.close()

            # Generate resource usage statistics
            stats = {
                'cpu': {
                    'max': df['cpu_percent'].max(),
                    'mean': df['cpu_percent'].mean(),
                    'p95': df['cpu_percent'].quantile(0.95),
                    'p99': df['cpu_percent'].quantile(0.99)
                },
                'memory': {
                    'max': df['memory_mb'].max(),
                    'mean': df['memory_mb'].mean(),
                    'p95': df['memory_mb'].quantile(0.95),
                    'p99': df['memory_mb'].quantile(0.99)
                }
            }
            
            with open(self.result_dir / "resource-stats.json", 'w') as f:
                json.dump(stats, f, indent=2)
                
        except Exception as e:
            print(f"Error analyzing resource usage: {e}")

    def analyze_collector_metrics(self):
        """Analyze OpenTelemetry Collector metrics"""
        metrics_file = self.result_dir / "collector-metrics.txt"
        if not metrics_file.exists():
            print("Warning: collector-metrics.txt not found")
            return

        try:
            # Parse collector metrics (this is a simplified version)
            # In a real implementation, you'd parse Prometheus metrics format
            with open(metrics_file) as f:
                metrics_data = f.read()
            
            # Extract batch send sizes (example parsing)
            batch_sizes = []
            for line in metrics_data.split('\n'):
                if 'otelcol_processor_batch_batch_send_size_sum' in line:
                    # Parse Prometheus metric format
                    # This is simplified - you'd use proper Prometheus parsing
                    try:
                        value = float(line.split()[-1])
                        batch_sizes.append(value)
                    except:
                        continue
            
            if batch_sizes:
                # Create throughput graph
                plt.figure(figsize=(10, 6))
                plt.plot(batch_sizes, 'b-', linewidth=2)
                plt.title('Collector Batch Send Sizes Over Time')
                plt.xlabel('Sample')
                plt.ylabel('Batch Size')
                plt.grid(True, alpha=0.3)
                plt.savefig(self.graphs_dir / "collector-throughput.png", dpi=300, bbox_inches='tight')
                plt.close()
                
        except Exception as e:
            print(f"Error analyzing collector metrics: {e}")

    def generate_summary_report(self):
        """Generate a comprehensive summary report"""
        if not self.test_result:
            return

        # Calculate test duration
        start_time = datetime.fromisoformat(self.test_result.start_time.replace('Z', '+00:00'))
        end_time = datetime.fromisoformat(self.test_result.end_time.replace('Z', '+00:00'))
        actual_duration = end_time - start_time

        # Load resource stats
        resource_stats = {}
        resource_stats_file = self.result_dir / "resource-stats.json"
        if resource_stats_file.exists():
            with open(resource_stats_file) as f:
                resource_stats = json.load(f)

        # Generate report
        report = f"""
# Stress Test Analysis Report

## Test Configuration
- **Test ID**: {self.test_result.test_id}
- **Scenario**: {self.test_result.scenario}
- **Target Spans/sec**: {self.test_result.spans_per_second:,}
- **Duration**: {self.test_result.duration}
- **CPU Limit**: {self.test_result.cpu_limit}
- **Memory Limit**: {self.test_result.memory_limit}

## Test Execution
- **Start Time**: {start_time.strftime('%Y-%m-%d %H:%M:%S UTC')}
- **End Time**: {end_time.strftime('%Y-%m-%d %H:%M:%S UTC')}
- **Actual Duration**: {actual_duration}

## Resource Usage Analysis
"""

        if resource_stats:
            report += f"""
### CPU Usage
- **Maximum**: {resource_stats['cpu']['max']:.1f}%
- **Average**: {resource_stats['cpu']['mean']:.1f}%
- **95th Percentile**: {resource_stats['cpu']['p95']:.1f}%
- **99th Percentile**: {resource_stats['cpu']['p99']:.1f}%

### Memory Usage
- **Maximum**: {resource_stats['memory']['max']:.0f} MB
- **Average**: {resource_stats['memory']['mean']:.0f} MB
- **95th Percentile**: {resource_stats['memory']['p95']:.0f} MB
- **99th Percentile**: {resource_stats['memory']['p99']:.0f} MB
"""

        # Check if any resource limits were exceeded
        violations = []
        if resource_stats:
            if resource_stats['cpu']['max'] > 80:
                violations.append(f"CPU usage exceeded 80% (max: {resource_stats['cpu']['max']:.1f}%)")
            if resource_stats['memory']['max'] > 1536:  # Example threshold
                violations.append(f"Memory usage was high (max: {resource_stats['memory']['max']:.0f} MB)")

        if violations:
            report += "\n## ⚠️ Resource Limit Violations\n"
            for violation in violations:
                report += f"- {violation}\n"
        else:
            report += "\n## ✅ All Resource Limits Respected\n"

        # Write report
        with open(self.result_dir / "analysis-report.md", 'w') as f:
            f.write(report)

        # Create a simple summary for quick viewing
        summary = f"""Test: {self.test_result.test_id}
Scenario: {self.test_result.scenario}
Target Rate: {self.test_result.spans_per_second:,} spans/sec
Duration: {actual_duration}
Max CPU: {resource_stats.get('cpu', {}).get('max', 'N/A')}%
Max Memory: {resource_stats.get('memory', {}).get('max', 'N/A')} MB
Status: {'⚠️ VIOLATIONS' if violations else '✅ PASSED'}
"""
        
        with open(self.result_dir / "analysis-summary.txt", 'w') as f:
            f.write(summary)

    def generate_recommendations(self):
        """Generate performance tuning recommendations"""
        if not self.test_result:
            return

        recommendations = []
        
        # Load resource stats
        resource_stats_file = self.result_dir / "resource-stats.json"
        if resource_stats_file.exists():
            with open(resource_stats_file) as f:
                resource_stats = json.load(f)
                
            cpu_max = resource_stats['cpu']['max']
            memory_max = resource_stats['memory']['max']
            
            # CPU recommendations
            if cpu_max > 90:
                recommendations.append("🔴 CPU usage is very high. Consider increasing CPU limits or reducing load.")
            elif cpu_max > 80:
                recommendations.append("🟡 CPU usage is high. Monitor for potential bottlenecks.")
            elif cpu_max < 30:
                recommendations.append("🟢 CPU usage is low. Could potentially handle higher load.")
                
            # Memory recommendations
            memory_limit_mb = float(self.test_result.memory_limit.replace('Gi', '')) * 1024
            memory_usage_percent = (memory_max / memory_limit_mb) * 100
            
            if memory_usage_percent > 90:
                recommendations.append("🔴 Memory usage is very high. Consider increasing memory limits.")
            elif memory_usage_percent > 80:
                recommendations.append("🟡 Memory usage is high. Monitor for potential memory pressure.")
            elif memory_usage_percent < 50:
                recommendations.append("🟢 Memory usage is low. Could potentially handle higher load.")
        
        # Capacity recommendations
        if self.test_result.spans_per_second >= 50000:
            recommendations.append("📈 High load test completed. Consider this as a capacity baseline.")
        elif self.test_result.spans_per_second >= 10000:
            recommendations.append("📊 Medium load test completed. Consider testing higher loads.")
        else:
            recommendations.append("📉 Low load test completed. Gradually increase load to find limits.")

        # Write recommendations
        with open(self.result_dir / "recommendations.md", 'w') as f:
            f.write("# Performance Recommendations\n\n")
            for rec in recommendations:
                f.write(f"- {rec}\n")

def main():
    parser = argparse.ArgumentParser(description='Analyze stress test results')
    parser.add_argument('result_dir', help='Path to results directory')
    parser.add_argument('--output-format', choices=['text', 'json', 'html'], default='text',
                      help='Output format for summary')
    args = parser.parse_args()

    if not os.path.exists(args.result_dir):
        print(f"Error: Result directory {args.result_dir} does not exist")
        sys.exit(1)

    analyzer = ResultAnalyzer(args.result_dir)
    analyzer.analyze()

if __name__ == "__main__":
    main()