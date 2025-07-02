#!/bin/bash

set -e

# Default values
SCENARIO=""
DURATION="10m"
SPANS_PER_SEC="1000"
CPU_LIMIT="1000m"
MEMORY_LIMIT="2Gi"
NAMESPACE="odigos-stress"
DEBUG="false"
RESULTS_DIR="./results"
CONFIG_DIR="./scenarios"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --scenario)
      SCENARIO="$2"
      shift 2
      ;;
    --duration)
      DURATION="$2"
      shift 2
      ;;
    --spans-per-sec)
      SPANS_PER_SEC="$2"
      shift 2
      ;;
    --cpu-limit)
      CPU_LIMIT="$2"
      shift 2
      ;;
    --memory-limit)
      MEMORY_LIMIT="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --debug)
      DEBUG="true"
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo "Options:"
      echo "  --scenario SCENARIO         Test scenario to run (required)"
      echo "  --duration DURATION         Test duration (default: 10m)"
      echo "  --spans-per-sec RATE         Spans per second (default: 1000)"
      echo "  --cpu-limit LIMIT           CPU limit (default: 1000m)"
      echo "  --memory-limit LIMIT        Memory limit (default: 2Gi)"
      echo "  --namespace NAMESPACE       Kubernetes namespace (default: odigos-stress)"
      echo "  --debug                     Enable debug output"
      echo "  --help                      Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

if [ -z "$SCENARIO" ]; then
  echo -e "${RED}Error: --scenario is required${NC}"
  exit 1
fi

# Generate unique test ID
TEST_ID=$(date +%Y%m%d_%H%M%S)_${SCENARIO}_${SPANS_PER_SEC}sps
RESULT_DIR="${RESULTS_DIR}/${TEST_ID}"

echo -e "${BLUE}=== Odigos Stress Testing Framework ===${NC}"
echo -e "${BLUE}Test ID: ${TEST_ID}${NC}"
echo -e "${BLUE}Scenario: ${SCENARIO}${NC}"
echo -e "${BLUE}Duration: ${DURATION}${NC}"
echo -e "${BLUE}Spans/sec: ${SPANS_PER_SEC}${NC}"
echo -e "${BLUE}CPU Limit: ${CPU_LIMIT}${NC}"
echo -e "${BLUE}Memory Limit: ${MEMORY_LIMIT}${NC}"
echo -e "${BLUE}Namespace: ${NAMESPACE}${NC}"
echo ""

# Create result directory
mkdir -p "${RESULT_DIR}"

# Function to log with timestamp
log() {
  echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${RESULT_DIR}/test.log"
}

# Function to cleanup on exit
cleanup() {
  log "${YELLOW}Cleaning up test environment...${NC}"
  
  # Save final collector logs
  kubectl logs -n odigos-system deployment/odigos-gateway --tail=1000 > "${RESULT_DIR}/collector-logs.txt" 2>/dev/null || true
  
  # Clean up test resources
  kubectl delete -f k8s/loadgen.yaml -n ${NAMESPACE} --ignore-not-found=true 2>/dev/null || true
  kubectl delete -f k8s/mock-backend.yaml -n ${NAMESPACE} --ignore-not-found=true 2>/dev/null || true
  
  log "${GREEN}Cleanup completed${NC}"
}

# Set trap for cleanup
trap cleanup EXIT

# Function to check if a pod is ready
wait_for_pod() {
  local pod_label=$1
  local namespace=$2
  local timeout=${3:-300}
  
  log "Waiting for pod with label ${pod_label} in namespace ${namespace}..."
  
  kubectl wait --for=condition=Ready \
    --timeout=${timeout}s \
    -n ${namespace} \
    pod -l ${pod_label}
}

# Function to get metrics from Prometheus
get_metrics() {
  local query=$1
  local start_time=$2
  local end_time=$3
  
  # This would typically query Prometheus API
  # For now, we'll use kubectl top and collector metrics
  echo "Getting metrics for query: ${query}"
}

# Load scenario configuration
SCENARIO_FILE="${CONFIG_DIR}/${SCENARIO}.yaml"
if [ ! -f "${SCENARIO_FILE}" ]; then
  log "${RED}Error: Scenario file ${SCENARIO_FILE} not found${NC}"
  exit 1
fi

log "${GREEN}Loading scenario configuration: ${SCENARIO_FILE}${NC}"

# Create namespace if it doesn't exist
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Apply monitoring resources (Prometheus, ServiceMonitor, etc.)
log "Setting up monitoring..."
kubectl apply -f monitoring/ -n ${NAMESPACE}

# Wait for monitoring to be ready
sleep 10

# Generate and apply load generator configuration
log "Generating load generator configuration..."
envsubst < k8s/loadgen.yaml.template > k8s/loadgen.yaml.tmp
sed -e "s/\${SPANS_PER_SEC}/${SPANS_PER_SEC}/g" \
    -e "s/\${DURATION}/${DURATION}/g" \
    -e "s/\${CPU_LIMIT}/${CPU_LIMIT}/g" \
    -e "s/\${MEMORY_LIMIT}/${MEMORY_LIMIT}/g" \
    -e "s/\${TEST_ID}/${TEST_ID}/g" \
    k8s/loadgen.yaml.tmp > k8s/loadgen.yaml

# Apply mock backend
log "Deploying mock backend..."
kubectl apply -f k8s/mock-backend.yaml -n ${NAMESPACE}
wait_for_pod "app=mock-backend" ${NAMESPACE}

# Apply load generator
log "Deploying load generator..."
kubectl apply -f k8s/loadgen.yaml -n ${NAMESPACE}
wait_for_pod "app=stress-loadgen" ${NAMESPACE}

# Record start time
START_TIME=$(date -u +%s)
log "${GREEN}Test started at $(date -u)${NC}"

# Monitor the test
log "Monitoring test progress..."
DURATION_SECONDS=$(echo ${DURATION} | sed 's/[^0-9]*//g')
if [[ ${DURATION} == *"m" ]]; then
  DURATION_SECONDS=$((DURATION_SECONDS * 60))
elif [[ ${DURATION} == *"h" ]]; then
  DURATION_SECONDS=$((DURATION_SECONDS * 3600))
fi

# Create monitoring script
cat > "${RESULT_DIR}/monitor.sh" << 'EOF'
#!/bin/bash

NAMESPACE=$1
DURATION_SECONDS=$2
RESULT_DIR=$3

# Function to get resource usage
get_resource_usage() {
  echo "$(date -u +%s),$(kubectl top pod -n odigos-system -l app.kubernetes.io/name=gateway --no-headers | awk '{print $2,$3}' | tr ' ' ',')" >> "${RESULT_DIR}/resource-usage.csv"
}

# Function to get collector metrics
get_collector_metrics() {
  # Get spans processed from collector metrics endpoint
  GATEWAY_POD=$(kubectl get pods -n odigos-system -l app.kubernetes.io/name=gateway -o jsonpath='{.items[0].metadata.name}')
  if [ ! -z "$GATEWAY_POD" ]; then
    kubectl exec -n odigos-system $GATEWAY_POD -- curl -s localhost:8888/metrics | grep "otelcol_processor_batch_batch_send_size_sum" >> "${RESULT_DIR}/collector-metrics.txt"
  fi
}

# Initialize CSV file
echo "timestamp,cpu,memory" > "${RESULT_DIR}/resource-usage.csv"

# Monitor for the duration of the test
END_TIME=$(($(date -u +%s) + DURATION_SECONDS))
while [ $(date -u +%s) -lt $END_TIME ]; do
  get_resource_usage
  get_collector_metrics
  sleep 5
done
EOF

chmod +x "${RESULT_DIR}/monitor.sh"

# Start monitoring in background
"${RESULT_DIR}/monitor.sh" ${NAMESPACE} ${DURATION_SECONDS} ${RESULT_DIR} &
MONITOR_PID=$!

# Wait for test completion
sleep ${DURATION_SECONDS}

# Stop monitoring
kill $MONITOR_PID 2>/dev/null || true

# Record end time
END_TIME=$(date -u +%s)
log "${GREEN}Test completed at $(date -u)${NC}"

# Collect final results
log "Collecting test results..."

# Get final resource usage
kubectl top pod -n odigos-system -l app.kubernetes.io/name=gateway --no-headers > "${RESULT_DIR}/final-resource-usage.txt"

# Get load generator logs
kubectl logs -n ${NAMESPACE} -l app=stress-loadgen --tail=1000 > "${RESULT_DIR}/loadgen-logs.txt"

# Get mock backend logs
kubectl logs -n ${NAMESPACE} -l app=mock-backend --tail=1000 > "${RESULT_DIR}/backend-logs.txt"

# Get collector configuration
kubectl get cm -n odigos-system odigos-gateway-conf -o yaml > "${RESULT_DIR}/collector-config.yaml"

# Generate summary
cat > "${RESULT_DIR}/summary.json" << EOF
{
  "testId": "${TEST_ID}",
  "scenario": "${SCENARIO}",
  "startTime": "${START_TIME}",
  "endTime": "${END_TIME}",
  "duration": "${DURATION}",
  "spansPerSecond": ${SPANS_PER_SEC},
  "cpuLimit": "${CPU_LIMIT}",
  "memoryLimit": "${MEMORY_LIMIT}",
  "namespace": "${NAMESPACE}"
}
EOF

# Generate basic report
log "Generating test report..."
python3 scripts/analyze-results.py "${RESULT_DIR}"

log "${GREEN}Stress test completed successfully!${NC}"
log "${GREEN}Results saved to: ${RESULT_DIR}${NC}"

# Display quick summary
if [ -f "${RESULT_DIR}/analysis-summary.txt" ]; then
  log "${YELLOW}=== Test Summary ===${NC}"
  cat "${RESULT_DIR}/analysis-summary.txt"
fi

echo ""
log "${BLUE}To view detailed results:${NC}"
log "${BLUE}  - Results directory: ${RESULT_DIR}${NC}"
log "${BLUE}  - Generate graphs: make stress-test-results${NC}"
log "${BLUE}  - View dashboard: make stress-test-dashboard${NC}"