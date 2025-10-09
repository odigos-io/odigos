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
  instance_type = "m6i.large"

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

  # ---------------- User data ----------------
    user_data = <<-BASH
    #!/bin/bash
    set -euxo pipefail
    exec > /var/log/user-data.log 2>&1

    yum install -y xfsprogs curl tar yum-utils

    # ---------------- Disk Setup ----------------
    for dev in /dev/nvme1n1 /dev/nvme2n1 /dev/nvme3n1; do
      for i in {1..120}; do
        [ -b "$dev" ] && break || sleep 2
      done
    done

    mkfs -t xfs /dev/nvme1n1
    mkdir -p /mnt/prometheus-data
    echo "/dev/nvme1n1 /mnt/prometheus-data xfs defaults,nofail 0 2" >> /etc/fstab
    mount /mnt/prometheus-data

    mkfs -t xfs /dev/nvme2n1
    mkdir -p /mnt/grafana-data
    echo "/dev/nvme2n1 /mnt/grafana-data xfs defaults,nofail 0 2" >> /etc/fstab
    mount /mnt/grafana-data

    mkfs -t xfs /dev/nvme3n1
    mkdir -p /mnt/clickhouse-data
    echo "/dev/nvme3n1 /mnt/clickhouse-data xfs defaults,nofail 0 2" >> /etc/fstab
    mount /mnt/clickhouse-data

    # ---------------- Prometheus ----------------
    cd /tmp
    curl -sSLo prometheus.tar.gz https://github.com/prometheus/prometheus/releases/download/v2.52.0/prometheus-2.52.0.linux-amd64.tar.gz
    tar -xzf prometheus.tar.gz
    install -m 0755 prometheus-2.52.0.linux-amd64/prometheus /usr/local/bin/prometheus

    id prometheus >/dev/null 2>&1 || useradd --no-create-home --shell /sbin/nologin prometheus
    mkdir -p /etc/prometheus /var/lib/prometheus
    chown -R prometheus:prometheus /etc/prometheus /var/lib/prometheus /mnt/prometheus-data

    cat >/etc/prometheus/prometheus.yml <<'EOF'
global:
  scrape_interval: 30s
scrape_configs: [] # receiver-only
EOF

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

    # ---------------- Grafana ----------------
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

    cat >/etc/grafana/grafana.ini <<'EOF'
[paths]
data = /mnt/grafana-data
[server]
http_addr = 127.0.0.1
http_port = 3000
EOF
    chown grafana:grafana /etc/grafana/grafana.ini

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

    # Create dashboard provisioning directory
    mkdir -p /etc/grafana/provisioning/dashboards
    cat >/etc/grafana/provisioning/dashboards/dashboard.yaml <<'EOF'
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
EOF



    chown -R grafana:grafana /etc/grafana/provisioning /mnt/grafana-data

    # Create script to import dashboard 15760 (Kubernetes Pods View)
    
    cat >/usr/local/bin/import-grafana-dashboard.sh <<'EOF'
#!/bin/bash
set -e

# Wait for Grafana to be ready
until curl -s http://127.0.0.1:3000/api/health > /dev/null; do
  sleep 2
done

# Wait for Grafana to fully initialize
sleep 10

# Download dashboard 15760 JSON
curl -s -L "https://grafana.com/api/dashboards/15760/revisions/latest/download" \
  -o /etc/grafana/provisioning/dashboards/kubernetes-pods-view.json

# Fix permissions
chown grafana:grafana /etc/grafana/provisioning/dashboards/kubernetes-pods-view.json

# Force Grafana to reload dashboards
curl -s -X POST http://127.0.0.1:3000/api/admin/provisioning/dashboards/reload || true

EOF
    
    chmod +x /usr/local/bin/import-grafana-dashboard.sh

    systemctl enable --now grafana-server

    # Import dashboard after Grafana starts
    nohup /usr/local/bin/import-grafana-dashboard.sh > /var/log/grafana-dashboard-import.log 2>&1 &

    # ---------------- ClickHouse ----------------
    rpm --import https://packages.clickhouse.com/rpm/stable/repodata/repomd.xml.key
    yum-config-manager --add-repo https://packages.clickhouse.com/rpm/clickhouse.repo
    yum install -y clickhouse-server clickhouse-client

    mkdir -p /mnt/clickhouse-data/{data,tmp,user_files,format_schemas,metadata,metadata_dropped,preprocessed_configs,flags,access}
    chown -R clickhouse:clickhouse /mnt/clickhouse-data
    chmod -R 750 /mnt/clickhouse-data
    chmod 755 /mnt/clickhouse-data/format_schemas

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

    mkdir -p /etc/clickhouse-server/users.d
    cat >/etc/clickhouse-server/users.d/default-password.xml <<'EOF'
<clickhouse>
  <users>
    <default>
      <password>stresstest</password>
    </default>
  </users>
</clickhouse>
EOF

    chown clickhouse:clickhouse /etc/clickhouse-server/users.d/default-password.xml
    chmod 640 /etc/clickhouse-server/users.d/default-password.xml

    chown root:root /etc/clickhouse-server/config.d/*.xml
    chmod 644 /etc/clickhouse-server/config.d/*.xml
    chmod 755 /etc/clickhouse-server/config.d
    chown clickhouse:clickhouse /etc/clickhouse-server/config.d

    systemctl daemon-reload
    systemctl enable clickhouse-server
    systemctl start clickhouse-server

    # ---------------- K6  ----------------
    cd /usr/local/bin
    curl -sSL https://github.com/grafana/k6/releases/download/v0.51.0/k6-v0.51.0-linux-amd64.tar.gz | tar xz
    mv k6-v0.51.0-linux-amd64/k6 /usr/bin/k6
    chmod +x /usr/bin/k6
    rm -rf k6-v0.51.0-linux-amd64

    mkdir -p /opt/k6/tests
    # Create simple dummy K6 load test script
    cat >/opt/k6/tests/loadtest.js <<'EOF'
import http from 'k6/http';

export const options = {
  vus: 1,
  duration: '30s',
};

export default function () {
  const url = 'http://localhost:8080/health';
  const response = http.get(url);
  console.log('Response status:', response.status);
}
EOF

    cat >/etc/systemd/system/k6-loadtest.service <<EOF
[Unit]
Description=K6 Load Test Runner
After=network-online.target

[Service]
# Environment variable for target service URL
# Set this to your application's URL when ready to test
# Example: Environment=K6_TARGET_SERVICE_URL=https://your-app.example.com
Environment=K6_TARGET_SERVICE_URL=
ExecStart=/usr/bin/k6 run /opt/k6/tests/loadtest.js
Restart=on-failure
WorkingDirectory=/opt/k6/tests

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable k6-loadtest
  BASH


  tags = { Name = "k6-runner" }


}
