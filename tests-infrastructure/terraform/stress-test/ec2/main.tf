########################################
# EC2 Prometheus Receiver + Grafana + ClickHouse + K6
########################################

# Data sources to get information from EKS deployment
data "terraform_remote_state" "eks" {
  backend = "local"
  config = {
    path = "../terraform.tfstate"
  }
}

# Get EKS node security group ID from remote state

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.88.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Read VPC/subnets + SG IDs from the EKS stack (apply EKS first)

# Base AMI (Amazon Linux 2)
data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["137112412989"] # Amazon

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ---------------- Security ----------------
resource "aws_security_group" "monitoring_ec2_sg" {
  name        = "monitoring-ec2-sg"
  description = "SG for Prometheus/Grafana/ClickHouse EC2"
  vpc_id      = data.terraform_remote_state.eks.outputs.vpc_id

  # HTTP/HTTPS for package downloads and API calls
  egress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  # DNS resolution
  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  # NTP for time synchronization
  egress {
    from_port   = 123
    to_port     = 123
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "monitoring-ec2-sg" }
}

# Prometheus remote_write from EKS nodes (9090)
resource "aws_security_group_rule" "in_9090_from_nodesg" {
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 9090
  to_port                  = 9090
  security_group_id        = aws_security_group.monitoring_ec2_sg.id
  source_security_group_id = data.terraform_remote_state.eks.outputs.node_security_group_id
  description              = "Prometheus remote_write from EKS nodes"
}


# ClickHouse HTTP (8123) from EKS nodes
resource "aws_security_group_rule" "in_8123_from_nodesg" {
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 8123
  to_port                  = 8123
  security_group_id        = aws_security_group.monitoring_ec2_sg.id
  source_security_group_id = data.terraform_remote_state.eks.outputs.node_security_group_id
  description              = "ClickHouse HTTP from EKS nodes"
}


# ClickHouse Native TCP (9000) from EKS nodes
resource "aws_security_group_rule" "in_9000_from_nodesg" {
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 9000
  to_port                  = 9000
  security_group_id        = aws_security_group.monitoring_ec2_sg.id
  source_security_group_id = data.terraform_remote_state.eks.outputs.node_security_group_id
  description              = "ClickHouse native TCP from EKS nodes"
}


# This rule was moved to the main EKS configuration


# ---------------- SSM for port-forwarding to UIs ----------------
resource "aws_iam_role" "ssm_core" {
  name               = "monitoring-ec2-ssm-core"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect    = "Allow",
      Principal = { Service = "ec2.amazonaws.com" },
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ssm_core_attach" {
  role       = aws_iam_role.ssm_core.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "ssm_core" {
  name = "monitoring-ec2-ssm-core"
  role = aws_iam_role.ssm_core.name
}

# ---------------- EC2 instance (attach data EBS at launch) ----------------
resource "aws_instance" "monitoring" {
  ami           = data.aws_ami.amazon_linux.id
  instance_type = "m6i.xlarge"

  subnet_id = data.terraform_remote_state.eks.outputs.private_subnet_ids[0]
  vpc_security_group_ids      = [aws_security_group.monitoring_ec2_sg.id]
  associate_public_ip_address = false

  iam_instance_profile = aws_iam_instance_profile.ssm_core.name

  # Root volume
  root_block_device {
    volume_size           = 20
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # Prometheus data volume -> will appear as /dev/nvme1n1 on Nitro
  ebs_block_device {
    device_name           = "/dev/sdf"
    volume_size           = 25
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # Grafana data volume -> will appear as /dev/nvme2n1 on Nitro
  ebs_block_device {
    device_name           = "/dev/sdg"
    volume_size           = 1
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # ClickHouse data volume -> will appear as /dev/nvme3n1 on Nitro
  ebs_block_device {
    device_name           = "/dev/sdh"
    volume_size           = 100
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # ---------------- User data ----------------
  user_data = base64encode(file("${path.module}/deploy-monitoring-infra.sh"))


  tags = { Name = "k6-runner" }


}
