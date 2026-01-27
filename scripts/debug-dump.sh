#!/bin/bash
#
# debug_dump.sh - Collects Odigos debug information into a tar.gz archive
#
# This script replicates the functionality of frontend/services/debug_dump.go
# using kubectl commands.
#
# Usage: ./debug_dump.sh [OPTIONS]
#
# Options:
#   -n, --namespace         Odigos system namespace (default: odigos-system)
#   -w, --include-workloads Include workload and pod YAMLs for each Source (default: false)
#   -f, --workload-namespaces  Comma-separated list of namespaces to collect workloads from
#                              (only used when --include-workloads is set, defaults to all namespaces)
#   -d, --dry-run           Show estimated file count and size without creating archive
#   -o, --output            Output file path (default: odigos-debug-{timestamp}.tar.gz)
#   -h, --help              Show this help message
#

set -e

# Default values
ODIGOS_NAMESPACE="${ODIGOS_NAMESPACE:-odigos-system}"
INCLUDE_WORKLOADS=false
WORKLOAD_NAMESPACES=""
DRY_RUN=false
OUTPUT_FILE=""
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters for dry-run
FILE_COUNT=0
TOTAL_SIZE=0

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Collects Odigos debug information into a tar.gz archive.

Options:
  -n, --namespace           Odigos system namespace (default: odigos-system)
  -w, --include-workloads   Include workload and pod YAMLs for each Source (default: false)
  -f, --workload-namespaces Comma-separated list of namespaces to collect workloads from
                            (only used when --include-workloads is set, defaults to all namespaces)
  -d, --dry-run             Show estimated file count and size without creating archive
  -o, --output              Output file path (default: odigos-debug-{timestamp}.tar.gz)
  -v, --verbose             Show tar contents after creation (for debugging)
  -h, --help                Show this help message

Examples:
  $0                                    # Basic collection from odigos-system namespace
  $0 -n my-odigos                       # Use custom odigos namespace
  $0 -w                                 # Include source workloads
  $0 -w -f "default,production"         # Include workloads from specific namespaces only
  $0 -d                                 # Dry run to estimate size
  $0 -o /tmp/debug.tar.gz               # Custom output file
  $0 -v                                 # Show tar structure after creation

EOF
    exit 0
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Format bytes to human-readable (portable, no bc required)
format_bytes() {
    local bytes=$1
    if [ "$bytes" -lt 1024 ]; then
        echo "${bytes} B"
    elif [ "$bytes" -lt 1048576 ]; then
        echo "$((bytes / 1024)) KB"
    elif [ "$bytes" -lt 1073741824 ]; then
        echo "$((bytes / 1048576)) MB"
    else
        echo "$((bytes / 1073741824)) GB"
    fi
}

# Check if kubectl is available
check_prerequisites() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi

    # Check if we can connect to the cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi

    # Check if namespace exists
    if ! kubectl get namespace "$ODIGOS_NAMESPACE" &> /dev/null; then
        log_error "Namespace '$ODIGOS_NAMESPACE' does not exist"
        exit 1
    fi
}

# Parse command-line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                ODIGOS_NAMESPACE="$2"
                shift 2
                ;;
            -w|--include-workloads)
                INCLUDE_WORKLOADS=true
                shift
                ;;
            -f|--workload-namespaces)
                WORKLOAD_NAMESPACES="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -o|--output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# Add file to tar or count for dry-run
add_file() {
    local filepath="$1"
    local content="$2"

    if [ "$DRY_RUN" = true ]; then
        local size=${#content}
        TOTAL_SIZE=$((TOTAL_SIZE + size))
        FILE_COUNT=$((FILE_COUNT + 1))
    else
        local dir
        dir=$(dirname "$filepath")
        mkdir -p "${WORK_DIR}/${dir}"
        # Use printf to avoid echo's interpretation of escape sequences and to not add extra newline
        printf '%s' "$content" > "${WORK_DIR}/${filepath}"
    fi
}

# Get resource YAML and clean managedFields
get_clean_yaml() {
    local resource_type="$1"
    local name="$2"
    local namespace="$3"

    local yaml_content
    if [ -n "$namespace" ]; then
        yaml_content=$(kubectl get "$resource_type" "$name" -n "$namespace" -o yaml 2>/dev/null || echo "")
    else
        yaml_content=$(kubectl get "$resource_type" "$name" -o yaml 2>/dev/null || echo "")
    fi

    if [ -z "$yaml_content" ]; then
        return 1
    fi

    # Remove managedFields using sed (cross-platform compatible)
    echo "$yaml_content" | sed '/^  managedFields:/,/^  [a-zA-Z]/{ /^  [a-zA-Z][a-zA-Z]*:/!d; /^  managedFields:/d; }'
}

# Collect pod logs
collect_pod_logs() {
    local namespace="$1"
    local pod_name="$2"
    local output_dir="$3"

    # Get containers in the pod
    local containers
    containers=$(kubectl get pod "$pod_name" -n "$namespace" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null)

    for container in $containers; do
        # Current logs
        local logs
        logs=$(kubectl logs "$pod_name" -n "$namespace" -c "$container" 2>&1 || echo "Error fetching logs")
        add_file "${output_dir}/pod-${pod_name}.${container}.logs" "$logs"

        # Check for previous logs (if container has been restarted)
        local restart_count
        restart_count=$(kubectl get pod "$pod_name" -n "$namespace" -o jsonpath="{.status.containerStatuses[?(@.name==\"${container}\")].restartCount}" 2>/dev/null || echo "0")
        if [ "$restart_count" -gt 0 ]; then
            local prev_logs
            prev_logs=$(kubectl logs "$pod_name" -n "$namespace" -c "$container" --previous 2>&1 || echo "Error fetching previous logs")
            add_file "${output_dir}/pod-${pod_name}.${container}.previous.logs" "$prev_logs"
        fi
    done

    # Get init containers
    local init_containers
    init_containers=$(kubectl get pod "$pod_name" -n "$namespace" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null)

    for container in $init_containers; do
        local logs
        logs=$(kubectl logs "$pod_name" -n "$namespace" -c "$container" 2>&1 || echo "Error fetching logs")
        add_file "${output_dir}/pod-${pod_name}.${container}.logs" "$logs"
    done
}

# Collect pods for a workload
collect_pods() {
    local namespace="$1"
    local output_dir="$2"
    local label_selector="$3"
    local include_logs="$4"

    local pods
    pods=$(kubectl get pods -n "$namespace" -l "$label_selector" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)

    for pod in $pods; do
        # Get pod YAML
        local yaml_content
        yaml_content=$(get_clean_yaml "pod" "$pod" "$namespace")
        if [ -n "$yaml_content" ]; then
            add_file "${output_dir}/pod-${pod}.yaml" "$yaml_content"
        fi

        # Collect logs if requested
        if [ "$include_logs" = true ]; then
            collect_pod_logs "$namespace" "$pod" "$output_dir"
        fi
    done
}

# Collect a workload (deployment, daemonset, statefulset)
collect_workload() {
    local namespace="$1"
    local name="$2"
    local kind="$3"
    local output_dir="$4"
    local include_logs="$5"

    local kind_lower
    kind_lower=$(echo "$kind" | tr '[:upper:]' '[:lower:]')

    log_info "Collecting $kind_lower/$name from namespace $namespace"

    # Get workload YAML
    local yaml_content
    yaml_content=$(get_clean_yaml "$kind_lower" "$name" "$namespace")
    if [ -z "$yaml_content" ]; then
        log_warn "Could not get $kind_lower $name"
        return
    fi
    add_file "${output_dir}/${kind_lower}-${name}.yaml" "$yaml_content"

    # Get label selector for pods
    local selector
    case "$kind_lower" in
        deployment|daemonset|statefulset)
            # Use go-template to extract label selector in key=value,key=value format (portable, no python needed)
            selector=$(kubectl get "$kind_lower" "$name" -n "$namespace" -o go-template='{{range $k, $v := .spec.selector.matchLabels}}{{$k}}={{$v}},{{end}}' 2>/dev/null | sed 's/,$//')
            ;;
        cronjob)
            # CronJobs are more complex - try to get jobs spawned by the cronjob
            selector="job-name"
            ;;
    esac

    if [ -n "$selector" ]; then
        collect_pods "$namespace" "$output_dir" "$selector" "$include_logs"
    fi
}

# Collect all odigos workloads from the odigos namespace
collect_odigos_workloads() {
    local root_dir="$1"

    log_info "Collecting Odigos workloads from namespace $ODIGOS_NAMESPACE"

    # Collect Deployments
    local deployments
    deployments=$(kubectl get deployments -n "$ODIGOS_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for name in $deployments; do
        collect_workload "$ODIGOS_NAMESPACE" "$name" "Deployment" "${root_dir}/${ODIGOS_NAMESPACE}/deployment-${name}" true
    done

    # Collect DaemonSets
    local daemonsets
    daemonsets=$(kubectl get daemonsets -n "$ODIGOS_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for name in $daemonsets; do
        collect_workload "$ODIGOS_NAMESPACE" "$name" "DaemonSet" "${root_dir}/${ODIGOS_NAMESPACE}/daemonset-${name}" true
    done

    # Collect StatefulSets
    local statefulsets
    statefulsets=$(kubectl get statefulsets -n "$ODIGOS_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for name in $statefulsets; do
        collect_workload "$ODIGOS_NAMESPACE" "$name" "StatefulSet" "${root_dir}/${ODIGOS_NAMESPACE}/statefulset-${name}" true
    done
}

# Collect source workloads (without logs)
collect_source_workloads() {
    local root_dir="$1"

    log_info "Collecting Source workloads"

    # Parse workload namespaces filter into a space-separated list
    local ns_filter=""
    if [ -n "$WORKLOAD_NAMESPACES" ]; then
        IFS=',' read -ra NS_ARRAY <<< "$WORKLOAD_NAMESPACES"
        for ns in "${NS_ARRAY[@]}"; do
            ns_filter="$ns_filter $(echo "$ns" | tr -d ' ')"
        done
    fi

    # Get all Sources using go-template (portable, no python needed)
    # Output format: namespace|name|kind per line
    local sources
    sources=$(kubectl get sources.odigos.io --all-namespaces -o go-template='{{range .items}}{{.spec.workload.namespace}}|{{.spec.workload.name}}|{{.spec.workload.kind}}{{"\n"}}{{end}}' 2>/dev/null || echo "")

    echo "$sources" | while IFS='|' read -r ns name kind; do
        # Skip empty lines and Namespace kind
        [ -z "$ns" ] || [ -z "$name" ] || [ -z "$kind" ] || [ "$kind" = "Namespace" ] && continue

        # Apply namespace filter if specified
        if [ -n "$ns_filter" ]; then
            if ! echo "$ns_filter" | grep -qw "$ns"; then
                continue
            fi
        fi

        local kind_lower
        kind_lower=$(echo "$kind" | tr '[:upper:]' '[:lower:]')
        collect_workload "$ns" "$name" "$kind" "${root_dir}/${ns}/${kind_lower}-${name}" false
    done
}

# Collect Odigos CRDs (portable, no python dependency)
collect_odigos_crds_simple() {
    local root_dir="$1"

    log_info "Collecting Odigos CRDs"

    # Get all API resources and filter for odigos.io groups
    local all_groups
    all_groups=$(kubectl api-versions 2>/dev/null | grep 'odigos.io' | cut -d'/' -f1 | sort -u)

    for group in $all_groups; do
        local group_resources
        group_resources=$(kubectl api-resources --api-group="$group" -o name 2>/dev/null || echo "")

        for resource in $group_resources; do
            # Get the short name (without the group suffix)
            local resource_name
            resource_name=$(echo "$resource" | cut -d'.' -f1)

            # Capitalize first letter for directory name (portable across macOS/Linux)
            local dir_name
            local first_char last_chars
            first_char=$(echo "$resource_name" | cut -c1 | tr '[:lower:]' '[:upper:]')
            last_chars=$(echo "$resource_name" | cut -c2-)
            dir_name="${first_char}${last_chars}"

            log_info "Collecting CRD: $resource_name"

            # Try to get items across all namespaces first
            local items
            items=$(kubectl get "$resource" --all-namespaces -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name --no-headers 2>/dev/null || \
                    kubectl get "$resource" -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name --no-headers 2>/dev/null || echo "")

            while read -r item_ns item_name; do
                [ -z "$item_name" ] && continue

                # For cluster-scoped resources, namespace will be <none>
                if [ "$item_ns" = "<none>" ] || [ -z "$item_ns" ]; then
                    item_ns="$ODIGOS_NAMESPACE"
                fi

                local yaml_content
                yaml_content=$(get_clean_yaml "$resource" "$item_name" "" 2>/dev/null || get_clean_yaml "$resource" "$item_name" "$item_ns" 2>/dev/null)

                if [ -n "$yaml_content" ]; then
                    add_file "${root_dir}/${item_ns}/${dir_name}/${item_name}.yaml" "$yaml_content"
                fi
            done <<< "$items"
        done
    done
}

# Collect ConfigMaps from odigos namespace
collect_configmaps() {
    local root_dir="$1"

    log_info "Collecting ConfigMaps from namespace $ODIGOS_NAMESPACE"

    local configmaps
    configmaps=$(kubectl get configmaps -n "$ODIGOS_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)

    for name in $configmaps; do
        local yaml_content
        yaml_content=$(get_clean_yaml "configmap" "$name" "$ODIGOS_NAMESPACE")
        if [ -n "$yaml_content" ]; then
            add_file "${root_dir}/${ODIGOS_NAMESPACE}/ConfigMaps/configmap-${name}.yaml" "$yaml_content"
        fi
    done
}

# Main function
main() {
    parse_args "$@"
    check_prerequisites

    # Generate timestamp and root directory name
    TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
    ROOT_DIR="odigos-debug-${TIMESTAMP}"

    if [ -z "$OUTPUT_FILE" ]; then
        OUTPUT_FILE="${ROOT_DIR}.tar.gz"
    fi

    log_info "Starting Odigos debug dump"
    log_info "Odigos namespace: $ODIGOS_NAMESPACE"
    log_info "Include workloads: $INCLUDE_WORKLOADS"
    if [ -n "$WORKLOAD_NAMESPACES" ]; then
        log_info "Workload namespaces filter: $WORKLOAD_NAMESPACES"
    fi

    if [ "$DRY_RUN" = true ]; then
        log_info "Running in dry-run mode"
    else
        # Create temporary working directory (avoid using TMPDIR as it's a system env var)
        WORK_DIR=$(mktemp -d)
        trap 'rm -rf "$WORK_DIR"' EXIT
    fi

    # Collect odigos workloads (with logs)
    collect_odigos_workloads "$ROOT_DIR"

    # Optionally collect source workloads (without logs)
    if [ "$INCLUDE_WORKLOADS" = true ]; then
        collect_source_workloads "$ROOT_DIR"
    fi

    # Collect Odigos CRDs
    collect_odigos_crds_simple "$ROOT_DIR"

    # Collect ConfigMaps
    collect_configmaps "$ROOT_DIR"

    if [ "$DRY_RUN" = true ]; then
        echo ""
        log_success "Dry run complete"
        echo "{"
        echo "  \"dryRun\": true,"
        echo "  \"includeWorkloads\": $INCLUDE_WORKLOADS,"
        echo "  \"workloadNamespaces\": \"$WORKLOAD_NAMESPACES\","
        echo "  \"fileCount\": $FILE_COUNT,"
        echo "  \"totalSizeBytes\": $TOTAL_SIZE,"
        echo "  \"totalSizeHuman\": \"$(format_bytes $TOTAL_SIZE)\""
        echo "}"
    else
        # Create tar.gz archive
        # The tar should contain paths starting with ROOT_DIR (e.g., odigos-debug-TIMESTAMP/namespace/...)
        # This matches the Go implementation which writes files with paths like "odigos-debug-TIMESTAMP/ns/file.yaml"
        # COPYFILE_DISABLE=1 prevents macOS from including AppleDouble (._*) metadata files
        log_info "Creating archive: $OUTPUT_FILE"
        COPYFILE_DISABLE=1 tar -czf "$OUTPUT_FILE" -C "$WORK_DIR" "$ROOT_DIR"

        local archive_size
        archive_size=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null || echo "unknown")

        log_success "Debug dump complete!"
        echo ""
        echo "Output file: $OUTPUT_FILE"
        echo "Archive size: $(format_bytes "$archive_size")"
        echo ""
        echo "To extract: tar -xzf $OUTPUT_FILE"

        # Show tar structure if verbose mode is enabled
        if [ "$VERBOSE" = true ]; then
            echo ""
            log_info "Tar structure (first 50 entries):"
            tar -tzf "$OUTPUT_FILE" | head -50
            echo ""
            log_info "Expected structure: $ROOT_DIR/<namespace>/<resource-type>/<files>"
        fi
    fi
}

main "$@"
