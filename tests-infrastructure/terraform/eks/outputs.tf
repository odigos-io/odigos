output "vpc_id" {
  description = "The VPC ID"
  value       = module.vpc.vpc_id
}

output "cluster_endpoint" {
  description = "The EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_name" {
  description = "The EKS cluster name"
  value       = module.eks.cluster_id
}
