# =============================================================================
# VPC OUTPUTS
# =============================================================================

output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "vpc_cidr" {
  description = "CIDR block of the VPC"
  value       = module.vpc.vpc_cidr_block
}

output "public_subnet_ids" {
  description = "List of IDs of public subnets"
  value       = module.vpc.public_subnets
}

output "private_subnet_ids" {
  description = "List of IDs of private subnets"
  value       = module.vpc.private_subnets
}

# =============================================================================
# EKS OUTPUTS
# =============================================================================

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.eks.cluster_security_group_id
}


output "node_security_group_id" {
  description = "Security group ID attached to the EKS node groups"
  value       = module.eks.node_security_group_id
}

# =============================================================================
# USEFUL COMMANDS
# =============================================================================

output "kubectl_config_command" {
  description = "Command to configure kubectl"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${var.cluster_name}"
}

output "cluster_info" {
  description = "Cluster information for reference"
  value = {
    name    = var.cluster_name
    region  = var.region
    version = var.cluster_version
  }
}

# =============================================================================
# PROMETHEUS DEPLOYMENT OUTPUTS
# =============================================================================

output "prometheus_deployment_status" {
  description = "Status of Prometheus deployment"
  value       = "Prometheus deployed via 3-step process: CRDs -> kube-prometheus-stack -> prometheus-agent"
  depends_on  = [null_resource.apply_prometheus_agent]
}

output "prometheus_namespace" {
  description = "Prometheus monitoring namespace"
  value       = "monitoring"
}

# =============================================================================
# ODIGOS DEPLOYMENT OUTPUTS
# =============================================================================

output "odigos_deployment_status" {
  description = "Status of Odigos deployment"
  value       = "Odigos installed, workload generators deployed, and ClickHouse destination configured"
  depends_on  = [null_resource.apply_odigos_clickhouse_destination]
}

output "workload_generators_status" {
  description = "Workload generators deployment status"
  value       = "Span generators deployed in load-test namespace"
  depends_on  = [null_resource.apply_workload_generators]
}

