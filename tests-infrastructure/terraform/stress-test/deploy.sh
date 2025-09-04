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
    local default_value="$2"
    
    # Try to read from terraform.tfvars first
    if [[ -f "terraform.tfvars" ]]; then
        local value=$(grep "^${var_name}" terraform.tfvars 2>/dev/null | cut -d'"' -f2)
        if [[ -n "$value" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    # Fall back to default value
    echo "$default_value"
}

# Run tofu command with error handling
run_tf() {
    local dir="$1"
    local command="$2"
    shift 2
    local args=("$@")
    
    log_info "Running: tofu $command ${args[*]} in $dir"
    
    if [[ "$dir" == "." ]]; then
        cd "$(dirname "$0")"
    else
        cd "$(dirname "$0")/$dir"
    fi
    
    if tofu "$command" "${args[@]}"; then
        log_success "Tofu $command completed successfully in $dir"
        return 0
    else
        log_error "Tofu $command failed in $dir"
        return 1
    fi
}


# Deploy infrastructure
deploy_infrastructure() {
    log_info "Starting infrastructure deployment..."
    
    # Validate prerequisites
    check_tofu
    check_aws
    
    # Deploy EKS cluster infrastructure only (creates VPC, no apps)
    log_info "Step 1: Deploying EKS cluster infrastructure..."
    if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=false" "-auto-approve"; then
        log_error "EKS infrastructure deployment failed"
        return 1
    fi
    
    # Deploy EC2 monitoring stack (depends on VPC from EKS)
    log_info "Step 2: Deploying EC2 monitoring stack..."
    if ! run_tf "ec2" "apply" "-auto-approve"; then
        log_error "EC2 deployment failed"
        return 1
    fi
    
    # Deploy Kubernetes applications (now that EC2 IP is available)
    log_info "Step 3: Deploying Kubernetes applications with EC2 IP..."
    if ! run_tf "." "apply" "-var=deploy_kubernetes_apps=true" "-auto-approve"; then
        log_error "Kubernetes applications deployment failed"
        return 1
    fi
    
    
    log_success "Infrastructure deployment completed!"
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
        
        # Step 1: Destroy EKS cluster (includes Kubernetes applications)
        log_info "Step 1: Destroying EKS cluster and Kubernetes applications..."
        if ! run_tf "." "destroy" "-var=deploy_kubernetes_apps=true" "-auto-approve"; then
            log_error "EKS destroy failed. Check tofu state and try again."
            return 1
        fi
        
        # Step 2: Destroy EC2 monitoring stack (after EKS is gone)
        log_info "Step 2: Destroying EC2 monitoring stack..."
        if ! run_tf "ec2" "destroy" "-auto-approve"; then
            log_error "EC2 destroy failed. This must succeed before EKS can be destroyed."
            log_info "EC2 instance is blocking VPC resource deletion."
            return 1
        fi
        
        
        log_success "Infrastructure destroyed successfully!"
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
        
        log_info "Cluster name: $cluster_name"
        log_info "Region: $region"
        
        # Check if cluster is actually running in AWS
        if aws eks describe-cluster --name "$cluster_name" --region "$region" &>/dev/null; then
            local status=$(aws eks describe-cluster --name "$cluster_name" --region "$region" --query 'cluster.status' --output text)
            log_success "Cluster status in AWS: $status"
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
    else
        log_warn "EC2 monitoring stack not found in state"
    fi
    
    # Check running instances
    log_info "Running EC2 Instances:"
    aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=k6-runner" "Name=instance-state-name,Values=running" \
        --query 'Reservations[].Instances[].[InstanceId,State.Name,Tags[?Key==`Name`].Value|[0]]' \
        --output table 2>/dev/null || log_warn "No running instances found"
    
    # Check Terraform-managed Kubernetes resources
    log_info "Terraform-managed Kubernetes Resources:"
    local k8s_resources=$(tofu state list 2>/dev/null | grep -E "(kubernetes_|kubectl_)" || echo "")
    if [[ -n "$k8s_resources" ]]; then
        log_success "Kubernetes resources managed by Terraform:"
        echo "$k8s_resources" | sed 's/^/  /'
    else
        log_warn "No Kubernetes resources found in Terraform state"
    fi
}

# Deploy only Kubernetes applications
deploy_apps() {
    log_info "Deploying Kubernetes applications..."
    
    # Check if EKS cluster exists
    if ! aws eks describe-cluster --name "$(get_tf_var cluster_name)" --region "$(get_tf_var region)" &>/dev/null; then
        log_error "EKS cluster not found. Please deploy infrastructure first with './deploy.sh deploy'"
        exit 1
    fi
    
    # Update kubeconfig
    log_info "Updating kubeconfig..."
    aws eks update-kubeconfig --region "$(get_tf_var region)" --name "$(get_tf_var cluster_name)"
    
    # Deploy Kubernetes applications
    log_info "Deploying Kubernetes applications with Terraform..."
    tofu apply -var="deploy_kubernetes_apps=true" -auto-approve
    
    log_success "Kubernetes applications deployed successfully!"
}

# Main script logic
main() {
    case "${1:-}" in
        "deploy")
            deploy_infrastructure
            ;;
        "apps")
            deploy_apps
            ;;
        "destroy")
            destroy_infrastructure
            ;;
        "status")
            check_status
            ;;
        *)
            echo "Usage: $0 {deploy|apps|destroy|status}"
            echo ""
            echo "Commands:"
            echo "  deploy  - Deploy the complete infrastructure (EKS + EC2 + Kubernetes apps)"
            echo "  apps    - Deploy only Kubernetes applications (requires EKS cluster)"
            echo "  destroy - Destroy the infrastructure with proper dependency order"
            echo "  status  - Check the current status of infrastructure"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
