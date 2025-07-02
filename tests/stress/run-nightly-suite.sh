#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RESULTS_BASE_DIR="./results/nightly"
DATE=$(date +%Y%m%d)
NIGHTLY_RESULTS_DIR="${RESULTS_BASE_DIR}/${DATE}"
SCENARIOS=(
    "basic-load-1k"
    "basic-load-5k"
    "basic-load-10k"
    "oom-prevention"
)

# Create results directory
mkdir -p "${NIGHTLY_RESULTS_DIR}"

# Function to log with timestamp
log() {
  echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${NIGHTLY_RESULTS_DIR}/nightly-suite.log"
}

# Function to send notification (placeholder for Slack/email integration)
send_notification() {
  local message="$1"
  local status="$2"
  
  # Placeholder for notification integration
  log "Notification: $message (Status: $status)"
  
  # Example Slack webhook (uncomment and configure if needed)
  # curl -X POST -H 'Content-type: application/json' \
  #   --data "{\"text\":\"$message\"}" \
  #   $SLACK_WEBHOOK_URL
}

# Function to run a single test scenario
run_scenario() {
  local scenario=$1
  local duration=${2:-"10m"}
  
  log "${BLUE}Starting scenario: ${scenario}${NC}"
  
  # Get scenario-specific configuration
  local spans_per_sec cpu_limit memory_limit
  case $scenario in
    "basic-load-1k")
      spans_per_sec=1000
      cpu_limit="1000m"
      memory_limit="2Gi"
      ;;
    "basic-load-5k")
      spans_per_sec=5000
      cpu_limit="1500m"
      memory_limit="3Gi"
      ;;
    "basic-load-10k")
      spans_per_sec=10000
      cpu_limit="2000m"
      memory_limit="4Gi"
      ;;
    "oom-prevention")
      spans_per_sec=15000
      cpu_limit="1000m"
      memory_limit="1Gi"
      duration="20m"
      ;;
    *)
      log "${RED}Unknown scenario: ${scenario}${NC}"
      return 1
      ;;
  esac
  
  # Run the test
  local test_start_time=$(date +%s)
  local test_result_dir
  
  if ./run-stress-test.sh \
    --scenario "$scenario" \
    --duration "$duration" \
    --spans-per-sec "$spans_per_sec" \
    --cpu-limit "$cpu_limit" \
    --memory-limit "$memory_limit" \
    --namespace "odigos-stress-nightly"; then
    
    log "${GREEN}Scenario ${scenario} completed successfully${NC}"
    
    # Find the most recent test result directory
    test_result_dir=$(find ./results -name "*${scenario}*" -type d | sort | tail -1)
    
    if [ -n "$test_result_dir" ]; then
      # Copy results to nightly directory
      cp -r "$test_result_dir" "${NIGHTLY_RESULTS_DIR}/"
      log "Results copied to ${NIGHTLY_RESULTS_DIR}"
    fi
    
    return 0
  else
    log "${RED}Scenario ${scenario} failed${NC}"
    return 1
  fi
}

# Function to generate comparative report
generate_comparative_report() {
  log "Generating comparative report..."
  
  cat > "${NIGHTLY_RESULTS_DIR}/comparative-report.md" << EOF
# Nightly Stress Test Report - ${DATE}

## Test Suite Summary

| Scenario | Status | Max CPU | Max Memory | Duration | Comments |
|----------|--------|---------|------------|----------|-----------|
EOF

  local total_tests=0
  local passed_tests=0
  
  for scenario in "${SCENARIOS[@]}"; do
    total_tests=$((total_tests + 1))
    
    # Find scenario result directory
    local scenario_dir=$(find "${NIGHTLY_RESULTS_DIR}" -name "*${scenario}*" -type d | head -1)
    
    if [ -d "$scenario_dir" ]; then
      local status="✅ PASSED"
      local max_cpu="N/A"
      local max_memory="N/A"
      local duration="N/A"
      local comments=""
      
      # Extract metrics from analysis
      if [ -f "$scenario_dir/resource-stats.json" ]; then
        max_cpu=$(jq -r '.cpu.max' "$scenario_dir/resource-stats.json" 2>/dev/null || echo "N/A")
        max_memory=$(jq -r '.memory.max' "$scenario_dir/resource-stats.json" 2>/dev/null || echo "N/A")
      fi
      
      if [ -f "$scenario_dir/summary.json" ]; then
        duration=$(jq -r '.duration' "$scenario_dir/summary.json" 2>/dev/null || echo "N/A")
      fi
      
      # Check for violations
      if [ -f "$scenario_dir/analysis-summary.txt" ]; then
        if grep -q "VIOLATIONS" "$scenario_dir/analysis-summary.txt"; then
          status="⚠️ VIOLATIONS"
          comments="Resource thresholds exceeded"
        else
          passed_tests=$((passed_tests + 1))
        fi
      else
        passed_tests=$((passed_tests + 1))
      fi
      
      echo "| $scenario | $status | ${max_cpu}% | ${max_memory} MB | $duration | $comments |" >> "${NIGHTLY_RESULTS_DIR}/comparative-report.md"
    else
      echo "| $scenario | ❌ FAILED | N/A | N/A | N/A | Test execution failed |" >> "${NIGHTLY_RESULTS_DIR}/comparative-report.md"
    fi
  done
  
  # Add summary section
  cat >> "${NIGHTLY_RESULTS_DIR}/comparative-report.md" << EOF

## Overall Results

- **Total Tests**: ${total_tests}
- **Passed**: ${passed_tests}
- **Failed**: $((total_tests - passed_tests))
- **Success Rate**: $(( (passed_tests * 100) / total_tests ))%

## Graphs and Detailed Results

Individual test results and graphs are available in the subdirectories:

EOF

  # List all result directories
  for result_dir in "${NIGHTLY_RESULTS_DIR}"/*/; do
    if [ -d "$result_dir" ]; then
      local dir_name=$(basename "$result_dir")
      echo "- [\`$dir_name\`](./$dir_name/)" >> "${NIGHTLY_RESULTS_DIR}/comparative-report.md"
    fi
  done
  
  # Generate HTML version if pandoc is available
  if command -v pandoc &> /dev/null; then
    pandoc "${NIGHTLY_RESULTS_DIR}/comparative-report.md" -o "${NIGHTLY_RESULTS_DIR}/comparative-report.html"
  fi
  
  log "Comparative report generated: ${NIGHTLY_RESULTS_DIR}/comparative-report.md"
}

# Function to check collector health before tests
check_collector_health() {
  log "Checking collector health..."
  
  # Check if Odigos is deployed
  if ! kubectl get deployment odigos-gateway -n odigos-system &>/dev/null; then
    log "${RED}Error: Odigos gateway not found. Please deploy Odigos first.${NC}"
    return 1
  fi
  
  # Check if gateway is ready
  if ! kubectl wait --for=condition=Available --timeout=60s deployment/odigos-gateway -n odigos-system; then
    log "${RED}Error: Odigos gateway is not ready${NC}"
    return 1
  fi
  
  log "${GREEN}Collector health check passed${NC}"
  return 0
}

# Function to cleanup before starting
cleanup_previous_tests() {
  log "Cleaning up previous test resources..."
  
  # Clean up any existing stress test resources
  kubectl delete namespace odigos-stress-nightly --ignore-not-found=true
  
  # Wait for cleanup
  sleep 10
  
  log "Cleanup completed"
}

# Main execution
main() {
  log "${BLUE}=== Starting Nightly Stress Test Suite ===${NC}"
  log "${BLUE}Date: ${DATE}${NC}"
  log "${BLUE}Results Directory: ${NIGHTLY_RESULTS_DIR}${NC}"
  
  # Pre-flight checks
  if ! check_collector_health; then
    send_notification "Nightly stress tests failed: Collector health check failed" "error"
    exit 1
  fi
  
  cleanup_previous_tests
  
  # Track overall results
  local suite_start_time=$(date +%s)
  local total_scenarios=${#SCENARIOS[@]}
  local successful_scenarios=0
  local failed_scenarios=()
  
  # Run each scenario
  for scenario in "${SCENARIOS[@]}"; do
    log "${YELLOW}Running scenario ${scenario} ($(( successful_scenarios + ${#failed_scenarios[@]} + 1 ))/${total_scenarios})${NC}"
    
    if run_scenario "$scenario"; then
      successful_scenarios=$((successful_scenarios + 1))
      log "${GREEN}✅ ${scenario} completed successfully${NC}"
    else
      failed_scenarios+=("$scenario")
      log "${RED}❌ ${scenario} failed${NC}"
    fi
    
    # Small delay between tests
    sleep 30
  done
  
  # Generate reports
  generate_comparative_report
  
  # Calculate suite duration
  local suite_end_time=$(date +%s)
  local suite_duration=$((suite_end_time - suite_start_time))
  local duration_hours=$((suite_duration / 3600))
  local duration_minutes=$(((suite_duration % 3600) / 60))
  
  # Final summary
  log "${BLUE}=== Nightly Test Suite Complete ===${NC}"
  log "${BLUE}Total Duration: ${duration_hours}h ${duration_minutes}m${NC}"
  log "${BLUE}Successful Scenarios: ${successful_scenarios}/${total_scenarios}${NC}"
  
  if [ ${#failed_scenarios[@]} -gt 0 ]; then
    log "${RED}Failed Scenarios: ${failed_scenarios[*]}${NC}"
    send_notification "Nightly stress tests completed with failures: ${failed_scenarios[*]}" "warning"
  else
    log "${GREEN}All scenarios completed successfully!${NC}"
    send_notification "Nightly stress tests completed successfully. All ${total_scenarios} scenarios passed." "success"
  fi
  
  # Cleanup
  cleanup_previous_tests
  
  log "${GREEN}Results available at: ${NIGHTLY_RESULTS_DIR}${NC}"
  
  # Return appropriate exit code
  if [ ${#failed_scenarios[@]} -gt 0 ]; then
    exit 1
  else
    exit 0
  fi
}

# Run with error handling
trap 'log "${RED}Nightly test suite interrupted${NC}"; cleanup_previous_tests; exit 130' INT TERM

main "$@"