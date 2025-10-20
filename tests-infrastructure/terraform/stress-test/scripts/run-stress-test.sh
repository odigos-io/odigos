#!/bin/bash

# Stress Test Orchestrator for OpenTelemetry Collector Throughput
# This script helps you run a complete stress test with throughput monitoring

set -e

# Default values
METRICS_URL="http://localhost:8888/metrics"
DURATION_MINUTES=1
INTERVAL_SECONDS=5
OUTPUT_FILE=""
NAMESPACE="load-test"
SPAN_GENERATOR_DEPLOYMENT="python-span-generator"
REPLICAS=3
SPAN_BYTES=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --metrics-url URL        Metrics endpoint URL (default: http://localhost:8888/metrics)"
    echo "  --duration MINUTES       Test duration in minutes (default: 5)"
    echo "  --interval SECONDS      Sampling interval in seconds (default: 5)"
    echo "  --output FILE           Output file for results (JSON format)"
    echo "  --namespace NAMESPACE   Kubernetes namespace for span generator (default: default)"
    echo "  --deployment NAME       Span generator deployment name"
    echo "  --replicas N            Number of replicas to scale the generator to (default: 3)"
    echo "  --span-bytes BYTES      Known per-span payload size in bytes (optional)"
    echo "  --help                  Show this help message"
    echo ""
    echo "Prerequisites:"
    echo "  1. Port-forward the collector pod: kubectl port-forward -n odigos-system <collector-pod> 8888:8888"
    echo "  2. Deploy span generators in your cluster"
    echo "  3. Install Python dependencies: pip install requests"
    echo ""
    echo "Example:"
    echo "  $0 --duration 10 --interval 3 --output results.json --namespace stress-test --deployment python-span-generator"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Python is available
    if ! command -v python3 &> /dev/null; then
        log_error "python3 is not installed or not in PATH"
        exit 1
    fi
    
    # Check if requests module is available
    if ! python3 -c "import requests" &> /dev/null; then
        log_error "Python requests module is not installed. Run: pip install requests"
        exit 1
    fi
    
    # Check if metrics endpoint is accessible
    log_info "Testing metrics endpoint: $METRICS_URL"
    if ! curl -s "$METRICS_URL" | grep -q "otelcol_exporter_sent_spans__spans__total"; then
        log_warning "Could not find otelcol_exporter_sent_spans__spans__total metric at $METRICS_URL"
        log_warning "Make sure you have port-forwarded the collector pod"
        log_warning "Continuing anyway..."
    else
        log_success "Metrics endpoint is accessible"
    fi
    
    log_success "Prerequisites check completed"
}

scale_span_generator() {
    if [ -n "$SPAN_GENERATOR_DEPLOYMENT" ]; then
        log_info "Scaling span generator deployment: $SPAN_GENERATOR_DEPLOYMENT"
        
        # Check if deployment exists
        if ! kubectl get deployment "$SPAN_GENERATOR_DEPLOYMENT" -n "$NAMESPACE" &> /dev/null; then
            log_error "Deployment '$SPAN_GENERATOR_DEPLOYMENT' not found in namespace '$NAMESPACE'"
            log_info "Available deployments in namespace '$NAMESPACE':"
            kubectl get deployments -n "$NAMESPACE" --no-headers | awk '{print "  - " $1}'
            exit 1
        fi
        
        # Scale up
        kubectl scale deployment "$SPAN_GENERATOR_DEPLOYMENT" --replicas=$REPLICAS -n "$NAMESPACE"
        log_success "Scaled up span generator to $REPLICAS replicas"
        
        # Wait for pods to be ready
        log_info "Waiting for span generator pods to be ready..."
        kubectl wait --for=condition=ready pod -l app="$SPAN_GENERATOR_DEPLOYMENT" -n "$NAMESPACE" --timeout=60s
        
        log_success "Span generator pods are ready"
    else
        log_warning "No span generator deployment specified. Make sure your span generators are running manually."
        log_info "Available deployments in namespace '$NAMESPACE':"
        kubectl get deployments -n "$NAMESPACE" --no-headers | awk '{print "  - " $1}'
    fi
}

run_stress_test() {
    log_info "Starting stress test monitoring..."
    log_info "Duration: $DURATION_MINUTES minutes"
    log_info "Sampling interval: $INTERVAL_SECONDS seconds"
    log_info "Metrics URL: $METRICS_URL"
    
    # Build command
    cmd="python3 monitor-throughput.py --metrics-url \"$METRICS_URL\" --duration $DURATION_MINUTES --interval $INTERVAL_SECONDS"
    
    if [ "$SPAN_BYTES" -gt 0 ]; then
        cmd="$cmd --span-bytes $SPAN_BYTES"
    fi
    
    if [ -n "$OUTPUT_FILE" ]; then
        cmd="$cmd --output \"$OUTPUT_FILE\""
    fi
    
    log_info "Running command: $cmd"
    
    # Run the monitor
    eval $cmd
}

cleanup() {
    log_info "Cleaning up..."
    
    if [ -n "$SPAN_GENERATOR_DEPLOYMENT" ]; then
        log_info "Scaling down span generator deployment: $SPAN_GENERATOR_DEPLOYMENT"
        if kubectl get deployment "$SPAN_GENERATOR_DEPLOYMENT" -n "$NAMESPACE" &> /dev/null; then
            kubectl scale deployment "$SPAN_GENERATOR_DEPLOYMENT" --replicas=0 -n "$NAMESPACE"
            log_success "Scaled down span generator to 0 replicas"
        else
            log_warning "Deployment '$SPAN_GENERATOR_DEPLOYMENT' not found, skipping cleanup"
        fi
    fi
    
    log_success "Cleanup completed"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --metrics-url)
            METRICS_URL="$2"
            shift 2
            ;;
        --duration)
            DURATION_MINUTES="$2"
            shift 2
            ;;
        --interval)
            INTERVAL_SECONDS="$2"
            shift 2
            ;;
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --deployment)
            SPAN_GENERATOR_DEPLOYMENT="$2"
            shift 2
            ;;
        --replicas)
            REPLICAS="$2"
            shift 2
            ;;
        --span-bytes)
            SPAN_BYTES="$2"
            shift 2
            ;;
        --help)
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    log_info "OpenTelemetry Collector Stress Test Orchestrator"
    log_info "================================================="
    
    check_prerequisites
    scale_span_generator
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Run the stress test
    run_stress_test
    
    log_success "Stress test completed successfully!"
    
    if [ -n "$OUTPUT_FILE" ] && [ -f "$OUTPUT_FILE" ]; then
        log_info "Results saved to: $OUTPUT_FILE"
        log_info "You can analyze the results with: cat $OUTPUT_FILE | jq ."
    fi
}

# Run main function
main
