terraform {
  required_version = ">= 1.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.88"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
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
    amd = "m6a.xlarge"
    arm = "m6g.xlarge"
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

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets  # Ensures at least 2 AZs

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

      tags = {
        AlwaysOn = "true"
      }
    }
  }
}

# VPC Endpoint for EKS API communication
resource "aws_vpc_endpoint" "eks_api" {
  vpc_id            = module.vpc.vpc_id
  service_name      = "com.amazonaws.${var.region}.eks"
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

# ---------------------------------------------------------------------------
# Destroy-time cleanup for orphaned k8s ELB security groups in this VPC
# (like: k8s-elb-a367a23ca7fae41879cf70630c16845b)
#
# NOTE:
# - We can only reference self.* in a destroy-time provisioner.
# - So we pass VPC ID + region via triggers and read them via self.triggers.*
# ---------------------------------------------------------------------------
resource "null_resource" "cleanup_non_default_sgs" {
  # Triggers capture the VPC and region so they are available as self.triggers.*
  triggers = {
    vpc_id = module.vpc.vpc_id
    region = var.region
  }

  # Important: this resource depends on the VPC so its *destroy* runs
  # before the VPC is destroyed (which is what we want).
  # The dependency is implied by triggers, so no explicit depends_on needed.

  provisioner "local-exec" {
    when = destroy

    command = <<-EOT
      set -euo pipefail

      VPC_ID="${self.triggers.vpc_id}"
      REGION="${self.triggers.region}"

      echo "Scanning for orphan k8s-elb-* security groups in VPC $VPC_ID (region: $REGION)..."

      # Only pick SGs created by Kubernetes ELB controller (k8s-elb-*)
      SG_IDS=$(aws ec2 describe-security-groups \
        --region "$REGION" \
        --filters "Name=vpc-id,Values=$${VPC_ID}" \
        --query "SecurityGroups[?starts_with(GroupName, 'k8s-elb-')].GroupId" \
        --output text || true)

      if [ -z "$${SG_IDS:-}" ]; then
        echo "No k8s-elb-* security groups found in VPC $VPC_ID"
        exit 0
      fi

      echo "Found Kubernetes ELB security groups to delete: $SG_IDS"

      for SG in $SG_IDS; do
        echo "Attempting to delete security group $SG..."

        # Clear ingress/egress rules defensively (sometimes helps)
        aws ec2 revoke-security-group-ingress \
          --region "$REGION" \
          --group-id "$SG" \
          --ip-permissions '[]' \
          >/dev/null 2>&1 || true

        aws ec2 revoke-security-group-egress \
          --region "$REGION" \
          --group-id "$SG" \
          --ip-permissions '[]' \
          >/dev/null 2>&1 || true

        aws ec2 delete-security-group \
          --region "$REGION" \
          --group-id "$SG" \
          >/dev/null 2>&1 || true
      done

      echo "Finished k8s-elb SG cleanup for VPC $VPC_ID"
    EOT

    interpreter = ["/bin/bash", "-c"]
  }
}
