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

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.19.0"

  name = "testing-vpc"
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

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.33.1"

  cluster_name    = var.cluster_name
  cluster_version = "1.32"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnets

  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true

  enable_cluster_creator_admin_permissions = true

  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64"
  }

  eks_managed_node_groups = {
    one = {
      name           = "node-group-1"
      instance_types = [var.node_spec]
      min_size       = var.node_count
      max_size       = var.node_count
      desired_size   = var.node_count
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_endpoint" "eks_api" {
  vpc_id            = module.vpc.vpc_id
  service_name      = "com.amazonaws.${var.region}.eks"
  vpc_endpoint_type = "Interface"
  subnet_ids        = module.vpc.private_subnets
  security_group_ids = [aws_security_group.eks_api_sg.id]
  private_dns_enabled = true

  depends_on = [module.eks]
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

  depends_on = [module.vpc]
}

resource "aws_security_group_rule" "allow_api_private" {
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = module.eks.cluster_security_group_id

  depends_on = [module.eks]
}

resource "aws_security_group_rule" "allow_4318_internal" {
  type              = "ingress"
  from_port         = 4318
  to_port           = 4318
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = module.eks.node_security_group_id
  description       = "Allow 4318 for e2e-tests-tempo"

  depends_on = [module.eks]
}

# Cleanup lingering ENIs
resource "null_resource" "cleanup_enis" {
  triggers = {
    vpc_id = module.vpc.vpc_id
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<EOT
      aws ec2 describe-network-interfaces --filters Name=vpc-id,Values=${self.triggers.vpc_id} --region ${var.region} --query 'NetworkInterfaces[].NetworkInterfaceId' --output text | xargs -n 1 -I {} aws ec2 delete-network-interface --network-interface-id {} --region ${var.region} || true
    EOT
  }

  depends_on = [module.eks, aws_vpc_endpoint.eks_api]
}

# Cleanup NAT Gateway and Elastic IP
resource "null_resource" "nat_gateway_cleanup" {
  triggers = {
    nat_gateway_id = module.vpc.natgw_ids[0]
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<EOT
      aws ec2 delete-nat-gateway --nat-gateway-id ${self.triggers.nat_gateway_id} --region ${var.region}
      aws ec2 wait nat-gateway-deleted --nat-gateway-ids ${self.triggers.nat_gateway_id} --region ${var.region} || true
      aws ec2 describe-addresses --filters Name=tag:aws:cloudformation:logical-id,Values=NatGateway --region ${var.region} --query 'Addresses[].AllocationId' --output text | xargs -n 1 aws ec2 release-address --allocation-id --region ${var.region} || true
    EOT
  }

  depends_on = [module.vpc]
}
