output "vpc_id" {
  value = module.vpc.vpc_id
}

output "public_subnet_ids" {
  value = module.vpc.public_subnets
}

output "private_subnet_ids" {
  value = module.vpc.private_subnets
}

output "eks_node_sg_id" {
  value = module.eks.node_security_group_id
}

output "eks_cluster_sg_id" {
  value = module.eks.cluster_security_group_id
}