########################################
# EC2 Prometheus Receiver + Grafana + ClickHouse (ephemeral)
########################################

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
data "terraform_remote_state" "eks" {
  backend = "local"
  config = {
    path = "../terraform.tfstate"
  }
}

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

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
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
  source_security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  description              = "Prometheus remote_write from EKS nodes"
}

# ClickHouse HTTP (8123) from EKS nodes
resource "aws_security_group_rule" "in_8123_from_nodesg" {
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 8123
  to_port                  = 8123
  security_group_id        = aws_security_group.monitoring_ec2_sg.id
  source_security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  description              = "ClickHouse HTTP from EKS nodes"
}

# ClickHouse Native TCP (9000) from EKS nodes
resource "aws_security_group_rule" "in_9000_from_nodesg" {
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 9000
  to_port                  = 9000
  security_group_id        = aws_security_group.monitoring_ec2_sg.id
  source_security_group_id = data.terraform_remote_state.eks.outputs.eks_node_sg_id
  description              = "ClickHouse native TCP from EKS nodes"
}

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
  instance_type = "m6i.large"

  subnet_id                   = data.terraform_remote_state.eks.outputs.public_subnet_ids[0]
  vpc_security_group_ids      = [aws_security_group.monitoring_ec2_sg.id]
  associate_public_ip_address = true

  iam_instance_profile = aws_iam_instance_profile.ssm_core.name

  # Root volume
  root_block_device {
    volume_size           = 40
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # Prometheus data volume -> will appear as /dev/nvme1n1 on Nitro
  ebs_block_device {
    device_name           = "/dev/sdf"
    volume_size           = 50
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # Grafana data volume -> will appear as /dev/nvme2n1 on Nitro
  ebs_block_device {
    device_name           = "/dev/sdg"
    volume_size           = 20
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

  user_data = <<-BASH
              #!/bin/bash
              set -euxo pipefail
              exec > >(tee /var/log/user-data.log | logger -t user-data -s 2>/dev/console) 2>&1

              # Install necessary utilities
              yum install -y xfsprogs curl tar yum-utils

              ## Disk Setup
              # Wait for EBS nvme devices to appear
              for dev in /dev/nvme1n1 /dev/nvme2n1 /dev/nvme3n1; do
                for i in {1..120}; do
                  [ -b "$dev" ] && break || sleep 2
                done
              done

              # Format, mount, and configure Prometheus data volume
              mkfs -t xfs /dev/nvme1n1
              mkdir -p /mnt/prometheus-data
              echo "/dev/nvme1n1 /mnt/prometheus-data xfs defaults,nofail 0 2" >> /etc/fstab
              mount /mnt/prometheus-data

              # Format, mount, and configure Grafana data volume
              mkfs -t xfs /dev/nvme2n1
              mkdir -p /mnt/grafana-data
              echo "/dev/nvme2n1 /mnt/grafana-data xfs defaults,nofail 0 2" >> /etc/fstab
              mount /mnt/grafana-data

              # Format, mount, and configure ClickHouse data volume
              mkfs -t xfs /dev/nvme3n1
              mkdir -p /mnt/clickhouse-data
              echo "/dev/nvme3n1 /mnt/clickhouse-data xfs defaults,nofail 0 2" >> /etc/fstab
              mount /mnt/clickhouse-data

              # ---------- Install and configure Prometheus ----------
              cd /tmp
              curl -sSLo prometheus.tar.gz https://github.com/prometheus/prometheus/releases/download/v2.52.0/prometheus-2.52.0.linux-amd64.tar.gz
              tar -xzf prometheus.tar.gz
              install -m 0755 prometheus-2.52.0.linux-amd64/prometheus /usr/local/bin/prometheus

              # Create user and set permissions
              id prometheus >/dev/null 2>&1 || useradd --no-create-home --shell /sbin/nologin prometheus
              mkdir -p /etc/prometheus /var/lib/prometheus
              chown -R prometheus:prometheus /etc/prometheus /var/lib/prometheus /mnt/prometheus-data

              # Create Prometheus configuration file
              cat >/etc/prometheus/prometheus.yml <<'EOF'
              global:
              scrape_interval: 30s
              scrape_configs: [] # receiver-only
              EOF

              # Create systemd service file for Prometheus
              cat >/etc/systemd/system/prometheus.service <<'EOF'
              [Unit]
              Description=Prometheus (remote-write receiver)
              After=network-online.target

              [Service]
              User=prometheus
              ExecStart=/usr/local/bin/prometheus \
                --config.file=/etc/prometheus/prometheus.yml \
                --web.enable-remote-write-receiver \
                --storage.tsdb.path=/mnt/prometheus-data \
                --storage.tsdb.retention.time=7d
              Restart=always
              RestartSec=5

              [Install]
              WantedBy=multi-user.target
              EOF
              systemctl daemon-reload
              systemctl enable --now prometheus

              # ---------- Install and configure Grafana ----------
              cat >/etc/yum.repos.d/grafana.repo <<'EOF'
              [grafana]
              name=Grafana OSS
              baseurl=https://packages.grafana.com/oss/rpm
              repo_gpgcheck=1
              enabled=1
              gpgcheck=1
              gpgkey=https://packages.grafana.com/gpg.key
              EOF
              yum install -y grafana

              # Configure Grafana to use the new data path and local binding
              cat >/etc/grafana/grafana.ini <<'EOF'
              [paths]
              data = /mnt/grafana-data
              [server]
              http_addr = 127.0.0.1
              http_port = 3000
              EOF
              chown grafana:grafana /etc/grafana/grafana.ini

              # Configure Grafana Prometheus datasource
              mkdir -p /etc/grafana/provisioning/datasources
              cat >/etc/grafana/provisioning/datasources/prometheus.yaml <<'EOF'
              apiVersion: 1
              datasources:
                - name: Prometheus
                  type: prometheus
                  access: proxy
                  url: http://127.0.0.1:9090
                  isDefault: true
                  jsonData:
                    httpMethod: POST
              EOF
              chown -R grafana:grafana /etc/grafana/provisioning /mnt/grafana-data

              systemctl enable --now grafana-server

             # ---------- Install and configure ClickHouse ----------
rpm --import https://packages.clickhouse.com/rpm/stable/repodata/repomd.xml.key
yum-config-manager --add-repo https://packages.clickhouse.com/rpm/clickhouse.repo
yum install -y clickhouse-server clickhouse-client

# STOP the service right after installation to prevent it from failing prematurely.
systemctl stop clickhouse-server || true

# Ensure data paths exist & owned
mkdir -p /mnt/clickhouse-data/{tmp,user_files,format_schemas}
chown -R clickhouse:clickhouse /mnt/clickhouse-data

# Safe overrides only
mkdir -p /etc/clickhouse-server/config.d
cat >/etc/clickhouse-server/config.d/01-paths.xml <<'EOF'
<clickhouse>
  <path>/mnt/clickhouse-data/</path>
  <tmp_path>/mnt/clickhouse-data/tmp/</tmp_path>
  <user_files_path>/mnt/clickhouse-data/user_files/</user_files_path>
  <format_schema_path>/mnt/clickhouse-data/format_schemas/</format_schema_path>
</clickhouse>
EOF

cat >/etc/clickhouse-server/config.d/02-network.xml <<'EOF'
<clickhouse>
  <listen_host>0.0.0.0</listen_host>
  <tcp_port>9000</tcp_port>
  <http_port>8123</http_port>
  <interserver_http_port>9012</interserver_http_port>
</clickhouse>
EOF

systemctl daemon-reload
systemctl enable --now clickhouse-server
            
  BASH

  tags = { Name = "k6-runner" }
}
