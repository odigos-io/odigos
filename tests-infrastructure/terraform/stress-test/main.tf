######## eks/main.tf ########
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

# --------------------------
# VPC
# --------------------------
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.19.0"

  name = "stress-testing-vpc"
  cidr = "10.0.0.0/16"
  azs  = slice(data.aws_availability_zones.available.names, 0, 2)

  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.3.0/24", "10.0.4.0/24"]

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

# --------------------------
# EKS cluster
# (disable KMS & CW logs to avoid AlreadyExists; also disable encryption config)
# --------------------------
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.33.1"

  cluster_name    = var.cluster_name
  cluster_version = "1.32"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true

  enable_cluster_creator_admin_permissions = true

  # Don't create/attach KMS or CW log group
  create_kms_key                = false
  create_cloudwatch_log_group   = false
  cluster_enabled_log_types     = []

  # IMPORTANT: make encryption config empty so the module doesn't expect provider_key_arn
  cluster_encryption_config = []

  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64"
  }

  eks_managed_node_groups = {
    one = {
      name           = "node-group-1"
      instance_types = [var.node_spec]

      min_size     = var.node_count
      max_size     = var.node_count
      desired_size = var.node_count
    }
  }
}

# --------------------------
# (Optional) VPC endpoint for EKS API
# --------------------------
resource "aws_vpc_endpoint" "eks_api" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.region}.eks"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.eks_api_sg.id]
  private_dns_enabled = true
}

resource "aws_security_group" "eks_api_sg" {
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# --------------------------
# SG rules for EC2 <-> EKS
# Pass EC2 SG via -var="monitoring_sg_id=sg-xxxxxxxx"
# --------------------------
resource "aws_security_group_rule" "allow_api_private" {
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = module.eks.cluster_security_group_id
}

# EKS nodes -> EC2 ClickHouse (HTTP 8123)
resource "aws_security_group_rule" "allow_clickhouse_from_eks_nodes" {
  type                     = "ingress"
  from_port                = 8123
  to_port                  = 8123
  protocol                 = "tcp"
  security_group_id        = var.monitoring_sg_id
  source_security_group_id = module.eks.node_security_group_id
  description              = "Allow EKS pods to send HTTP traffic to ClickHouse"
}

# EKS nodes -> EC2 ClickHouse (native TCP 9000)
resource "aws_security_group_rule" "allow_clickhouse_from_eks_nodes_tcp" {
  type                     = "ingress"
  from_port                = 9000
  to_port                  = 9000
  protocol                 = "tcp"
  security_group_id        = var.monitoring_sg_id
  source_security_group_id = module.eks.node_security_group_id
  description              = "Allow EKS pods to send TCP traffic to ClickHouse"
}

# EC2 Prometheus -> EKS node-exporter (9100)
resource "aws_security_group_rule" "allow_prometheus_to_node_exporter" {
  type                     = "ingress"
  from_port                = 9100
  to_port                  = 9100
  protocol                 = "tcp"
  security_group_id        = module.eks.node_security_group_id
  source_security_group_id = var.monitoring_sg_id
  description              = "Allow EC2 Prometheus to scrape node-exporter"
}

