terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.88"
    }
  }
}

data "aws_availability_zones" "available" {}

locals {
  # Map the architecture to the correct EKS AMI type
  ami_type_map = {
    amd = "AL2_x86_64"
    arm = "AL2_ARM_64"
  }

  # Sensible defaults for each arch (may be overridden by var.node_spec)
  default_instance_map = {
    amd = "m6a.large"
    arm = "m6g.large"
  }

  # Resolve final values
  resolved_ami_type      = local.ami_type_map[var.platform]
  resolved_instance_type = coalesce(var.node_spec, local.default_instance_map[var.platform])
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.19.0"

  name = "testing-vpc"
  cidr = "10.0.0.0/16"
  azs  = slice(data.aws_availability_zones.available.names, 0, 2) # Ensure at least 2 AZs

  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"] # At least 2 different AZs
  public_subnets  = ["10.0.3.0/24", "10.0.4.0/24"]  # Public subnets

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.33.1"

  cluster_name    = var.cluster_name
  cluster_version = "1.32"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnets  # Ensures at least 2 AZs

  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true

  enable_cluster_creator_admin_permissions = true

  eks_managed_node_group_defaults = {
    ami_type = local.resolved_ami_type
  }

  eks_managed_node_groups = {
    one = {
      name           = "node-group-1"
      instance_types = [local.resolved_instance_type]

      min_size     = var.node_count
      max_size     = var.node_count
      desired_size = var.node_count
    }
  }
}


# VPC Endpoint for EKS API communication
resource "aws_vpc_endpoint" "eks_api" {
  vpc_id          = module.vpc.vpc_id
  service_name    = "com.amazonaws.${var.region}.eks"
  vpc_endpoint_type = "Interface"

  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.eks_api_sg.id]
  private_dns_enabled = true
}

# Security Group for EKS API communication
resource "aws_security_group" "eks_api_sg" {
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"] # Your VPC CIDR
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}


# Allow Kubernetes API to communicate with private services
resource "aws_security_group_rule" "allow_api_private" {
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = module.eks.cluster_security_group_id
}


# Allow port 4318 for internal communication to simple-trace-db service
resource "aws_security_group_rule" "allow_4318_internal" {
  type              = "ingress"
  from_port         = 4318
  to_port           = 4318
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"] # Your VPC CIDR
  security_group_id = module.eks.node_security_group_id
  description       = "Allow 4318 for e2e-tests-tempo"
}

# EKS LB cleanup (runs BEFORE EKS is destroyed)
resource "null_resource" "cleanup_lb" {
  triggers = {
    cluster_name = var.cluster_name
    region       = var.region         # put region into triggers so we can use self.triggers.region
  }

  # Run this at destroy-time; only reference self.triggers.*
  provisioner "local-exec" {
    when    = destroy
    command = <<EOT
      # Try to connect to the cluster (ok if it's already gone)
      aws eks update-kubeconfig --name ${self.triggers.cluster_name} --region ${self.triggers.region} || true

      # Ask Kubernetes to remove any Services of type LoadBalancer (so k8s cleans up AWS LBs/ENIs)
      # If kubectl is unavailable or cluster is gone, this will just no-op.
      if command -v kubectl >/dev/null 2>&1; then
        kubectl get svc -A -o json \
          | jq -r '.items[] | select(.spec.type=="LoadBalancer") | [.metadata.namespace, .metadata.name] | @tsv' \
          | while IFS=$'\t' read -r ns name; do
              echo "Deleting LB Service: $ns/$name"
              kubectl delete svc "$name" -n "$ns" --wait=true --ignore-not-found=true || true
            done
      fi

      # Give AWS a moment to detach ENIs from deleted LBs
      sleep 60
    EOT
  }

  # Ensure this runs BEFORE the EKS module is destroyed (destroy order is reverse of create)
  depends_on = [module.eks]
}
