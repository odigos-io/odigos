provider "aws" {
  region = "us-east-1"
}

# Read VPC/subnets + SG IDs from the EKS stack (apply EKS first)
data "terraform_remote_state" "eks" {
  backend = "local"
  config = {
    path = "../terraform.tfstate"  # adjust if your EKS state is elsewhere
  }
}

# Latest Amazon Linux 2 (AL2)
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

# Subnet object to get the AZ for the EBS volumes
data "aws_subnet" "public0" {
  id = data.terraform_remote_state.eks.outputs.public_subnet_ids[0]
}

# ----- Add rules ON the existing EKS node SG so EC2 is usable -----
# Your IP -> SSH
resource "aws_security_group_rule" "allow_ssh_from_your_ip" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["212.117.136.162/32"]
  description       = "SSH to EC2 (instance uses EKS node SG)"
}

# Your IP -> Grafana
resource "aws_security_group_rule" "allow_grafana_from_your_ip" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 3000
  to_port           = 3000
  protocol          = "tcp"
  cidr_blocks       = ["212.117.136.162/32"]
  description       = "Grafana UI to EC2"
}

# Your IP -> Prometheus UI
resource "aws_security_group_rule" "allow_prom_ui_from_your_ip" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 9090
  to_port           = 9090
  protocol          = "tcp"
  cidr_blocks       = ["212.117.136.162/32"]
  description       = "Prometheus UI to EC2"
}

# VPC -> ClickHouse HTTP (8123)
resource "aws_security_group_rule" "allow_clickhouse_http_from_vpc" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 8123
  to_port           = 8123
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  description       = "VPC to EC2 ClickHouse HTTP"
}

# VPC -> ClickHouse native (9000)
resource "aws_security_group_rule" "allow_clickhouse_tcp_from_vpc" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 9000
  to_port           = 9000
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  description       = "VPC to EC2 ClickHouse native"
}

# VPC -> OTLP HTTP (optional)
resource "aws_security_group_rule" "allow_otlp_from_vpc" {
  type              = "ingress"
  security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  from_port         = 4318
  to_port           = 4318
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  description       = "VPC to EC2 OTLP HTTP"
}

# ----- EC2 instance (attach the EKS node SG) -----
resource "aws_instance" "monitoring" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "m6i.large"
  subnet_id              = data.terraform_remote_state.eks.outputs.public_subnet_ids[0]
  vpc_security_group_ids = [data.terraform_remote_state.eks.outputs.eks_node_sg_id]
  key_name               = var.key_pair_name

  associate_public_ip_address = true

  root_block_device {
    volume_size = 100
    volume_type = "gp3"
  }

  tags = { Name = "monitoring-node" }
}

# EBS volumes in the SAME AZ as the chosen subnet/instance
resource "aws_ebs_volume" "prometheus_data" {
  availability_zone = data.aws_subnet.public0.availability_zone
  size              = 50
  type              = "gp3"
  tags = { Name = "prometheus-data-volume" }
}

resource "aws_ebs_volume" "grafana_data" {
  availability_zone = data.aws_subnet.public0.availability_zone
  size              = 20
  type              = "gp3"
  tags = { Name = "grafana-data-volume" }
}

# Attachments
resource "aws_volume_attachment" "prometheus_attachment" {
  device_name  = "/dev/sdf"
  volume_id    = aws_ebs_volume.prometheus_data.id
  instance_id  = aws_instance.monitoring.id
  force_detach = true
}

resource "aws_volume_attachment" "grafana_attachment" {
  device_name  = "/dev/sdg"
  volume_id    = aws_ebs_volume.grafana_data.id
  instance_id  = aws_instance.monitoring.id
  force_detach = true
}
