#!/bin/bash

# Odigos Stress Test Infrastructure Deployment Script
# Handles EKS cluster, EC2 monitoring stack, and Kubernetes applications via Terraform

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Check if tofu is installed
check_tofu() {
    if ! command -v tofu &> /dev/null; then
        log_error "Tofu is not installed. Please install it first."
        exit 1
    fi
}

# Check if AWS CLI is configured
check_aws() {
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS CLI is not configured. Please run 'aws configure' first."
        exit 1
    fi
}

# Get Terraform variable value from terraform.tfvars or variables.tf defaults
get_tf_var() {
    local var_name="$1"
    local default_value="${2:-}"
    
    # Try to read from terraform.tfvars first
    if [[ -f "terraform.tfvars" ]]; then
        local value=$(grep "^${var_name}" terraform.tfvars 2>/dev/null | cut -d'"' -f2)
        if [[ -n "$value" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    # Fall back to default value (empty string if not provided)
    echo "$default_value"
}

# Check if Terraform variable is properly declared
check_tf_variable() {
    local var_name="$1"
    
    # Check if variable is declared in variables.tf
    if grep -q "variable \"${var_name}\"" variables.tf 2>/dev/null; then
        return 0
    else
        log_error "Variable '$var_name' is not declared in variables.tf"
        return 1
    fi
}

# Clean up Terraform state if corrupted
cleanup_tf_state() {
    log_info "Cleaning up Terraform state..."
    
    # Remove any lock files
    rm -f .terraform.lock.hcl
    rm -rf .terraform/
    
    # Reinitialize
    if run_tf "." "init" "-upgrade"; then
        log_success "Terraform state cleaned up successfully"
        return 0
    else
        log_error "Failed to clean up Terraform state"
        return 1
    fi
}

# Run tofu command with error handling
run_tf() {
    local dir="$1"
    local command="$2"
    shift 2
    local args=("$@")
    
    # Store original directory
    local original_dir="$(pwd)"
    
    # Handle empty args array
    if [[ ${#args[@]} -eq 0 ]]; then
        log_info "Running: tofu $command in $dir"
    else
        log_info "Running: tofu $command ${args[*]} in $dir"
    fi
    
    if [[ "$dir" == "." ]]; then
        cd "$(dirname "$0")"
    else
        # Verify the directory exists before trying to cd
        local target_dir="$(dirname "$0")/$dir"
        if [[ ! -d "$target_dir" ]]; then
            log_error "Directory $target_dir does not exist"
            return 1
        fi
        cd "$target_dir"
    fi
    
    # Set timeout for long-running operations (if timeout command is available)
    local timeout_cmd=""
    if [[ "$command" == "destroy" && -n "$(command -v timeout 2>/dev/null)" ]]; then
        timeout_cmd="timeout 30m"  # 30 minute timeout for destroy operations
        log_info "Setting 30-minute timeout for destroy operation..."
    fi
    
    if [[ ${#args[@]} -eq 0 ]]; then
        if [[ -n "$timeout_cmd" ]]; then
            $timeout_cmd tofu "$command"
        else
            tofu "$command"
        fi
    else
        if [[ -n "$timeout_cmd" ]]; then
            $timeout_cmd tofu "$command" "${args[@]}"
        else
            tofu "$command" "${args[@]}"
        fi
    fi
    
    local exit_code=$?
    
    # Return to original directory
    cd "$original_dir"
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "Tofu $command completed successfully in $dir"
        return 0
    elif [[ $exit_code -eq 124 ]]; then
        log_warn "Tofu $command timed out in $dir (30 minutes)"
        return 1
    else
        log_error "Tofu $command failed in $dir (exit code: $exit_code)"
        return 1
    fi
}


# Deploy infrastructure
deploy_infrastructure() {
    log_info "Starting infrastructure deployment..."
    
    # Validate prerequisites
    check_tofu
    check_aws
    
    # Initialize Terraform to ensure all variables and providers are properly set up
    log_info "Initializing Terraform configuration..."
    if ! run_tf "." "init" "-upgrade"; then
        log_error "Terraform initialization failed"
        return 1
    fi
    
    # Validate the configuration to ensure variables are properly declared
    log_info "Validating Terraform configuration..."
    if ! run_tf "." "validate"; then
        log_error "Terraform configuration validation failed"
        return 1
    fi
    
    # Deploy EKS cluster infrastructure only (creates VPC, no apps)
    log_info "Step 1: Deploying EKS cluster infrastructure..."
    if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=false" "-auto-approve"; then
        log_error "EKS cluster infrastructure deployment failed (VPC, EKS cluster, node groups)"
        return 1
    fi
    
    # Deploy EC2 monitoring stack (depends on VPC from EKS)
    log_info "Step 2: Deploying EC2 monitoring stack..."
    log_info "Initializing EC2 Terraform configuration..."
    if ! run_tf "ec2" "init"; then
        log_error "EC2 Terraform initialization failed"
        return 1
    fi
    
    if ! run_tf "ec2" "apply" "-auto-approve"; then
        log_error "EC2 monitoring stack deployment failed (monitoring instance, ClickHouse, Grafana)"
        return 1
    fi
    
    # Deploy Kubernetes applications (now that EC2 IP is available)
    log_info "Step 3: Deploying Kubernetes applications with EC2 IP..."
    
    # Configure kubectl to use the correct cluster endpoint
    local cluster_name=$(tofu output -raw cluster_name 2>/dev/null || get_tf_var cluster_name "odigos-stress-test")
    local region=$(tofu output -raw region 2>/dev/null || get_tf_var region "us-east-1")
    log_info "Configuring kubectl for cluster: $cluster_name"
    aws eks update-kubeconfig --region "$region" --name "$cluster_name"
    
    # Re-initialize and validate before deploying k8s apps to ensure clean state
    log_info "Re-initializing Terraform for Kubernetes applications..."
    if ! run_tf "." "init" "-upgrade"; then
        log_error "Terraform re-initialization failed"
        return 1
    fi
    
    if ! run_tf "." "validate"; then
        log_error "Terraform validation failed before app deployment"
        return 1
    fi
    
    if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=true" "-var=deploy_load_test_apps=true" "-auto-approve"; then
        log_error "Kubernetes applications deployment failed (Odigos, Prometheus, ClickHouse destination, load-test apps)"
        return 1
    fi
    
    
    log_success "Infrastructure deployment completed!"
    
    # Display monitoring access commands
    local instance_id=$(cd ec2 && tofu output -raw monitoring_instance_id 2>/dev/null || echo "Unknown")
    if [[ "$instance_id" != "Unknown" && "$instance_id" != "None" ]]; then
        log_info "Monitoring Access Commands:"
        echo "  • ClickHouse (port 8123):"
        echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"8123\"],\"localPortNumber\":[\"8123\"]}'"
        echo "  • Prometheus (port 9090):"
        echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"9090\"],\"localPortNumber\":[\"9090\"]}'"
        echo "  • Grafana (port 3000):"
        echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"3000\"],\"localPortNumber\":[\"3000\"]}'"
    else
        log_warn "EC2 monitoring instance not found or not running"
    fi
}




# Destroy infrastructure with proper dependency handling
destroy_infrastructure() {
    log_warn "This will destroy all infrastructure. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        log_info "Starting destruction process..."
        
        # Validate prerequisites
        check_tofu
        check_aws
        
        # Initialize Terraform to ensure all variables and providers are properly set up
        log_info "Initializing Terraform configuration..."
        if ! run_tf "." "init"; then
            log_error "Terraform initialization failed"
            return 1
        fi
        
        # Step 1: Destroy EC2 monitoring stack first (to free up VPC resources)
        log_info "Step 1: Destroying EC2 monitoring stack..."
        log_info "Initializing EC2 Terraform configuration..."
        if ! run_tf "ec2" "init"; then
            log_error "EC2 Terraform initialization failed"
            return 1
        fi
        if ! run_tf "ec2" "destroy" "-auto-approve"; then
            log_error "EC2 monitoring stack destroy failed (monitoring instance, ClickHouse, Grafana). Both EKS and EC2 states must exist for proper destruction."
            return 1
        fi
        
        # Step 2: Destroy EKS cluster (includes Kubernetes applications and VPC)
        log_info "Step 2: Destroying EKS cluster and Kubernetes applications..."
        log_warn "This may take 20+ minutes due to EKS node group deletion..."
        log_info "Progress: This step will destroy Kubernetes apps, EKS cluster, and VPC resources"
        
        if ! run_tf "." "destroy" "-auto-approve"; then
            log_warn "EKS destroy encountered issues, but this is often due to network connectivity during verification."
            log_info "Checking if resources were actually destroyed..."
            
            # Check if the cluster still exists in AWS
            local cluster_name=$(tofu output -raw cluster_name 2>/dev/null || echo "unknown")
            if [[ "$cluster_name" != "unknown" ]]; then
                if aws eks describe-cluster --name "$cluster_name" --region "$(tofu output -raw region 2>/dev/null || echo "us-east-1")" &>/dev/null; then
                    log_error "EKS cluster still exists in AWS. Manual cleanup may be required."
                    log_info "You can manually delete the cluster with: aws eks delete-cluster --name $cluster_name --region $(tofu output -raw region 2>/dev/null || echo "us-east-1")"
                    return 1
                else
                    log_success "EKS cluster successfully destroyed (verification failed due to network issues)"
                fi
            else
                log_success "EKS cluster successfully destroyed"
            fi
        fi
        
        # Final verification
        log_info "Verifying destruction completion..."
        local remaining_resources=$(tofu state list 2>/dev/null | wc -l)
        if [[ $remaining_resources -eq 0 ]]; then
            log_success "Infrastructure destroyed successfully! All resources removed from Terraform state."
        else
            log_warn "Some resources may still exist in Terraform state. Run './deploy.sh status' to check."
        fi
    else
        log_info "Destruction cancelled."
    fi
}


# Check status
check_status() {
    log_info "Checking infrastructure status..."
    
    # Check EKS cluster
    log_info "EKS Cluster Status:"
    local state_check=$(tofu state list 2>/dev/null | grep "module.eks.aws_eks_cluster" || echo "NOT_FOUND")
    if [[ "$state_check" != "NOT_FOUND" ]]; then
        log_success "EKS cluster exists in state"
        
        # Get cluster info from tofu output
        local cluster_info=$(tofu output -json cluster_info 2>/dev/null || echo '{}')
        local cluster_name=$(echo "$cluster_info" | jq -r '.name // "Unknown"')
        local region=$(echo "$cluster_info" | jq -r '.region // "us-east-1"')
        local version=$(echo "$cluster_info" | jq -r '.version // "Unknown"')
        
        log_info "Cluster name: $cluster_name"
        log_info "Region: $region"
        log_info "Kubernetes version: $version"
        
        # Check if cluster is actually running in AWS
        if aws eks describe-cluster --name "$cluster_name" --region "$region" &>/dev/null; then
            local status=$(aws eks describe-cluster --name "$cluster_name" --region "$region" --query 'cluster.status' --output text)
            local endpoint=$(aws eks describe-cluster --name "$cluster_name" --region "$region" --query 'cluster.endpoint' --output text)
            local created=$(aws eks describe-cluster --name "$cluster_name" --region "$region" --query 'cluster.createdAt' --output text)
            
            log_success "Cluster status in AWS: $status"
            log_info "API endpoint: $endpoint"
            log_info "Created: $created"
            
            # Check kubectl context
            log_info "Kubernetes Context:"
            local current_context=$(kubectl config current-context 2>/dev/null || echo "Not configured")
            if [[ "$current_context" == *"$cluster_name"* ]]; then
                log_success "kubectl context: $current_context"
            else
                log_warn "kubectl context: $current_context (may need 'aws eks update-kubeconfig')"
            fi
            
            # Check cluster health and pods
            log_info "Cluster Health:"
            if kubectl cluster-info &>/dev/null; then
                log_success "Cluster API server: accessible"
                
                # Check node status
                local node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
                local ready_nodes=$(kubectl get nodes --no-headers 2>/dev/null | grep " Ready " | wc -l)
                log_info "Nodes: $ready_nodes/$node_count ready"
                
                # Check system pods
                local system_pods=$(kubectl get pods -n kube-system --no-headers 2>/dev/null | wc -l)
                local running_system_pods=$(kubectl get pods -n kube-system --no-headers 2>/dev/null | grep " Running " | wc -l)
                log_info "System pods: $running_system_pods/$system_pods running"
                
            else
                log_warn "Cluster API server: not accessible (may need kubeconfig update)"
            fi
        else
            log_warn "Cluster not found in AWS (may be creating or destroyed)"
        fi
    else
        log_warn "EKS cluster not found in state"
    fi
    
    # Check EC2 monitoring stack
    log_info "EC2 Monitoring Stack Status:"
    if tofu state list -state=ec2/terraform.tfstate 2>/dev/null | grep -q "aws_instance"; then
        log_success "EC2 monitoring stack exists in state"
        
        # Get EC2 instance details
        local instance_id=$(cd ec2 && tofu output -raw monitoring_instance_id 2>/dev/null || echo "Unknown")
        local private_ip=$(cd ec2 && tofu output -raw monitoring_instance_private_ip 2>/dev/null || echo "Unknown")
        log_info "Instance ID: $instance_id"
        log_info "Private IP: $private_ip"
        
        # Store instance_id for later use in monitoring access commands
        export EC2_INSTANCE_ID="$instance_id"
        
        # Display monitoring access commands
        if [[ "$instance_id" != "Unknown" && "$instance_id" != "None" ]]; then
            log_info "Monitoring Access Commands:"
            echo "  • Grafana (port 3000):"
            echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"3000\"],\"localPortNumber\":[\"3000\"]}'"
            echo "  • ClickHouse (port 8123):"
            echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"8123\"],\"localPortNumber\":[\"8123\"]}'"
            echo "  • Prometheus (port 9090):"
            echo "    aws ssm start-session --target $instance_id --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"9090\"],\"localPortNumber\":[\"9090\"]}'"
        else
            log_warn "EC2 monitoring instance not found or not running"
        fi
    else
        log_warn "EC2 monitoring stack not found in state"
    fi
}

# Deploy only Kubernetes applications
deploy_k8s_apps() {
    log_info "Deploying Kubernetes applications..."
    
    # Check if EKS cluster exists
    local cluster_name=$(tofu output -raw cluster_name 2>/dev/null || get_tf_var cluster_name "odigos-stress-test")
    local region=$(tofu output -raw region 2>/dev/null || get_tf_var region "us-east-1")
    
    if ! aws eks describe-cluster --name "$cluster_name" --region "$region" &>/dev/null; then
        log_error "EKS cluster not found. Please deploy infrastructure first with './deploy.sh deploy'"
        exit 1
    fi
    
    # Configure kubectl to use the correct cluster endpoint
    log_info "Configuring kubectl for cluster: $cluster_name"
    aws eks update-kubeconfig --region "$region" --name "$cluster_name"
    
    # Check if critical components are actually installed, if not, force re-deployment
    log_info "Checking if Kubernetes applications are properly installed..."
    
    local needs_taint=false
    
    # Check Prometheus CRDs
    if ! kubectl get crd | grep -q "prometheuses.monitoring.coreos.com"; then
        log_warn "Prometheus CRDs not found, will force re-deployment..."
        tofu taint 'null_resource.install_prometheus_crds' 2>/dev/null || true
        needs_taint=true
    fi
    
    # Check Prometheus stack
    if ! kubectl get pods -n monitoring 2>/dev/null | grep -q "prometheus"; then
        log_warn "Prometheus stack not found, will force re-deployment..."
        tofu taint 'null_resource.install_kube_prometheus_stack[0]' 2>/dev/null || true
        tofu taint 'null_resource.apply_prometheus_agent[0]' 2>/dev/null || true
        needs_taint=true
    fi
    
    # Check Odigos (simplified - just check if namespace exists)
    if ! kubectl get namespace odigos-system 2>/dev/null | grep -q "odigos-system"; then
        log_warn "Odigos namespace not found, will force re-deployment..."
        tofu taint 'null_resource.install_odigos[0]' 2>/dev/null || true
        tofu taint 'null_resource.apply_odigos_clickhouse_destination[0]' 2>/dev/null || true
        needs_taint=true
    fi
    
    # Check ClickHouse destination
    if ! kubectl get destination clickhouse-destination -n odigos-system 2>/dev/null | grep -q "clickhouse-destination"; then
        log_warn "ClickHouse destination not found, will force re-deployment..."
        tofu taint 'null_resource.apply_odigos_clickhouse_destination[0]' 2>/dev/null || true
        needs_taint=true
    fi
    
    # Check Odigos sources (only if load-test apps are being deployed)
    if [[ "$1" == "--with-load-test" ]]; then
        # Check workload generators
        if ! kubectl get pods -n load-test 2>/dev/null | grep -q "span-generator"; then
            log_warn "Workload generators not found, will force re-deployment..."
            tofu taint 'null_resource.apply_workload_generators[0]' 2>/dev/null || true
            needs_taint=true
        fi
        
        # Check Odigos sources
        if ! kubectl get sources -n load-test 2>/dev/null | grep -q "span-generator-source"; then
            log_warn "Odigos sources not found, will force re-deployment..."
            tofu taint 'null_resource.apply_odigos_sources[0]' 2>/dev/null || true
            needs_taint=true
        fi
    fi
    
    if [[ "$needs_taint" == "true" ]]; then
        log_info "Some components missing, will re-deploy them..."
    else
        log_info "All components are already installed, skipping re-deployment"
    fi
    
    # Initialize Terraform to ensure all variables and providers are properly set up
    log_info "Initializing Terraform configuration..."
    if ! run_tf "." "init"; then
        log_error "Terraform initialization failed for k8s apps deployment"
        return 1
    fi
    
    # Kubeconfig is already updated by the main script
    
    # Deploy Kubernetes applications
    log_info "Deploying Kubernetes applications with Terraform..."
    
    # Check if load-test apps flag is provided
    if [[ "$1" == "--with-load-test" ]]; then
        log_info "Including load-test applications..."
        if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=true" "-var=deploy_load_test_apps=true" "-auto-approve"; then
            log_error "Kubernetes applications deployment failed (with load-test apps)"
            return 1
        fi
    else
        log_info "Deploying core applications only..."
        if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=true" "-var=deploy_load_test_apps=false" "-auto-approve"; then
            log_error "Kubernetes applications deployment failed (core apps only)"
            return 1
        fi
    fi
    
    log_success "Kubernetes applications deployed successfully!"
}

# Main script logic
main() {
    case "${1:-}" in
        "deploy")
            deploy_infrastructure
            ;;
        "k8s-apps")
            deploy_k8s_apps "${2:-}"
            ;;
        "destroy")
            destroy_infrastructure
            ;;
        "status")
            check_status
            ;;
        *)
            echo "Usage: $0 {deploy|k8s-apps|destroy|status}"
            echo ""
            echo "Commands:"
            echo "  deploy  - Deploy the complete infrastructure (EKS + EC2 + Kubernetes apps + load-test workloads)"
            echo "  k8s-apps [--with-load-test] - Deploy only Kubernetes applications (requires EKS cluster)"
            echo "    --with-load-test  - Also deploy load-test applications"
            echo "  destroy - Destroy the infrastructure with proper dependency order"
            echo "  status  - Check the current status of infrastructure"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
