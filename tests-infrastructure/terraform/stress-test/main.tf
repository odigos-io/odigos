terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.88"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.14"
    }
  }
}

data "aws_availability_zones" "available" {}

locals {
  name_prefix = var.cluster_name
  common_tags = {
    Environment = "stress-test"
    Project     = "odigos"
    ManagedBy   = "terraform"
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.19.0"

  name = "${local.name_prefix}-vpc"
  cidr = var.vpc_cidr
  azs  = slice(data.aws_availability_zones.available.names, 0, var.availability_zones_count)

  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets

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
  version = "=20.33.1"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnets

  cluster_endpoint_private_access = true
  cluster_endpoint_public_access = true
  # cluster_endpoint_public_access_cidrs = ["<IP_ADDRESS>/32"]

  enable_cluster_creator_admin_permissions = true

  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64" # Amazon Linux 2 (x86-64)
  }

  eks_managed_node_groups = {
    stress-test = {
      name           = "stress-test-nodes"
      instance_types = [var.node_spec]
      min_size       = var.node_min_size
      desired_size   = var.node_desired_size
      max_size       = var.node_max_size
      disk_size      = var.node_disk_size
    }
  }
}

# VPC Endpoint for EKS API communication (Kubernetes API server)
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
    cidr_blocks = ["10.0.0.0/16"] 
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
  cidr_blocks       = [var.vpc_cidr]
  security_group_id = module.eks.cluster_security_group_id
}

# Allow K6 monitoring instance to reach pods via AWS Network Load Balancer
# This will be created when both EKS and EC2 are deployed
resource "aws_security_group_rule" "allow_k6_to_nlb" {
  count = var.create_monitoring_connection ? 1 : 0
  
  type                     = "ingress"
  from_port                = 8080
  to_port                  = 8080
  protocol                 = "tcp"
  security_group_id        = module.eks.cluster_security_group_id
  source_security_group_id = data.aws_security_group.monitoring_ec2[0].id
  description              = "Allow k6 EC2 to reach pods via NLB"
}

# Find the monitoring EC2 security group by name
data "aws_security_group" "monitoring_ec2" {
  count = var.create_monitoring_connection ? 1 : 0
  name  = "monitoring-ec2-sg"
}

# Configure kubectl provider
provider "kubectl" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
  token                  = data.aws_eks_cluster_auth.cluster.token
  load_config_file       = false
}

# Get EKS cluster auth token
data "aws_eks_cluster_auth" "cluster" {
  name = module.eks.cluster_name
}

# Get EC2 monitoring instance IP from remote state
# Data sources to get information from EC2 deployment
data "terraform_remote_state" "ec2" {
  backend = "local"
  config = {
    path = "./ec2/terraform.tfstate"
  }
}

# =============================================================================
# PROMETHEUS DEPLOYMENT (3-step process)
# =============================================================================

# Step 1: Install Prometheus Operator CRDs (only if deploying apps)
resource "null_resource" "install_prometheus_crds" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  
  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Wait for cluster to be ready
      kubectl wait --for=condition=Ready nodes --all --timeout=300s
      
      # Install Prometheus Operator CRDs
      echo "Installing Prometheus Operator CRDs..."
      helm upgrade --install prometheus-crds prometheus-community/prometheus-operator-crds
      
      echo "Prometheus CRDs installation completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
  }
}

# Step 2: Install kube-prometheus-stack (only if deploying apps)
resource "null_resource" "install_kube_prometheus_stack" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  depends_on = [null_resource.install_prometheus_crds[0]]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Install kube-prometheus-stack with values file
      echo "Installing kube-prometheus-stack..."
      helm upgrade -i kube-prometheus-stack prometheus-community/kube-prometheus-stack \
        -n monitoring \
        -f ${path.module}/deploy/monitoring-stack/prometheus-values.yaml \
        --create-namespace \
        --wait
      
      echo "kube-prometheus-stack installation completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    prometheus_values = filesha256("${path.module}/deploy/monitoring-stack/prometheus-values.yaml")
  }
}

# Step 3: Apply prometheus-agent.yaml (only if deploying apps)
resource "null_resource" "apply_prometheus_agent" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  depends_on = [null_resource.install_kube_prometheus_stack[0]]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Wait for CRDs to be ready
      kubectl wait --for condition=established --timeout=60s crd/prometheusagents.monitoring.coreos.com
      kubectl wait --for condition=established --timeout=60s crd/servicemonitors.monitoring.coreos.com
      
      # Ensure monitoring namespace exists
      kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
      
      # Apply prometheus-agent.yaml with dynamic EC2 IP
      echo "Applying prometheus-agent.yaml..."
      kubectl apply -f - <<EOF
${templatefile("${path.module}/deploy/monitoring-stack/prometheus-agent.yaml", {
  ec2_ip = try(data.terraform_remote_state.ec2.outputs.monitoring_instance_private_ip, "pending")
})}
EOF
      
      echo "Prometheus Agent deployment completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    prometheus_agent = filesha256("${path.module}/deploy/monitoring-stack/prometheus-agent.yaml")
    ec2_ip = try(data.terraform_remote_state.ec2.outputs.monitoring_instance_private_ip, "pending")
  }
}

# Step 4: Deploy Odigos (only if deploying apps)
resource "null_resource" "install_odigos" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  depends_on = [null_resource.apply_prometheus_agent[0]]

  provisioner "local-exec" {
    command = <<-EOT
      set -e  # Exit on any error
      
      # Update kubeconfig
      echo "Updating kubeconfig..."
      aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}
      
      # Install Odigos using Helm
      echo "Installing Odigos with Helm..."
      
      # Add Odigos Helm repository
      echo "Adding Odigos Helm repository..."
      helm repo add odigos https://odigos-io.github.io/odigos || echo "Repository already exists"
      helm repo update
      
      # Install Odigos in the cluster
      echo "Installing Odigos in the cluster..."
      helm upgrade --install odigos odigos/odigos \
        --namespace odigos-system \
        --create-namespace \
        --set image.tag=${var.odigos_tag} \
        --set onprem-token=${var.odigos_api_key}
      
      # Note: Helm --wait flag already ensures Odigos is ready
      
      # Verify installation
      echo "Verifying Odigos installation..."
      kubectl get pods -n odigos-system
      
      echo "Odigos installation completed successfully!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    odigos_version = "latest"
    force_reinstall = "force-odigos-reinstall-$(date +%s)"
  }
}

# Step 5: Deploy Workload Generators (only if deploying load-test apps)
resource "null_resource" "apply_workload_generators" {
  count = var.deploy_load_test_apps ? 1 : 0
  depends_on = [null_resource.install_odigos[0]]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Create load-test namespace first (idempotent)
      echo "Ensuring load-test namespace exists..."
      if ! kubectl get namespace load-test >/dev/null 2>&1; then
        kubectl create namespace load-test
      else
        echo "load-test namespace already exists"
      fi
      
      # Apply workload generators from individual deployment files
      echo "Applying workload generators from generators directory..."
      kubectl apply -f ${path.module}/deploy/workloads/generators/go/deployment.yaml
      kubectl apply -f ${path.module}/deploy/workloads/generators/java/deployment.yaml
      kubectl apply -f ${path.module}/deploy/workloads/generators/node/deployment.yaml
      kubectl apply -f ${path.module}/deploy/workloads/generators/python/deployment.yaml
      
      # Deployments applied, no waiting required
      
      echo "Workload generators deployment completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    go_generator = filesha256("${path.module}/deploy/workloads/generators/go/deployment.yaml")
    java_generator = filesha256("${path.module}/deploy/workloads/generators/java/deployment.yaml")
    node_generator = filesha256("${path.module}/deploy/workloads/generators/node/deployment.yaml")
    python_generator = filesha256("${path.module}/deploy/workloads/generators/python/deployment.yaml")
    namespace_fix = "load-test-namespace-created"
  }
}

# Step 6: Apply Odigos Sources (only if deploying apps)
resource "null_resource" "apply_odigos_sources" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  depends_on = [
    null_resource.install_odigos[0]
  ]

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Apply Odigos sources only if workload generators are deployed
      if [[ "${var.deploy_load_test_apps}" == "true" ]]; then
        # Wait for odigos-instrumentor deployment to be ready before applying sources
        echo "Waiting for odigos-instrumentor deployment to be ready..."
        kubectl wait --for=condition=available --timeout=120s deployment/odigos-instrumentor -n odigos-system
        
        # Apply Odigos sources for workload generators
        echo "Applying Odigos sources for workload generators..."
        kubectl apply -f ${path.module}/deploy/odigos/sources.yaml
      else
        echo "Skipping Odigos sources (no workload generators deployed)"
      fi
      
      echo "Odigos sources deployment completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    odigos_sources = filesha256("${path.module}/deploy/odigos/sources.yaml")
    namespace_fix = "load-test-namespace-created"
  }
}

# Step 7: Deploy Odigos ClickHouse Destination (only if deploying apps)
resource "null_resource" "apply_odigos_clickhouse_destination" {
  count = var.deploy_kubernetes_apps ? 1 : 0
  depends_on = [
    null_resource.install_odigos[0],
    null_resource.apply_odigos_sources[0],
    data.terraform_remote_state.ec2
  ]

  provisioner "local-exec" {
    command = <<-EOT
      set -e  # Exit on any error
      
      # Update kubeconfig
      echo "Updating kubeconfig..."
      aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}
      
      # Verify Odigos is running (it should be ready due to dependency)
      echo "Verifying Odigos is running..."
      kubectl get pods -n odigos-system
      
      # Get EC2 IP from Terraform remote state
      EC2_IP="${data.terraform_remote_state.ec2.outputs.monitoring_instance_private_ip}"
      
      # Verify EC2 IP is available
      if [[ -z "$EC2_IP" || "$EC2_IP" == "destroyed" ]]; then
        echo "ERROR: EC2 monitoring instance IP not available. Please deploy EC2 stack first."
        exit 1
      fi
      
      # Apply ClickHouse destination with dynamic EC2 IP
      echo "Applying ClickHouse destination with EC2 IP: $EC2_IP"
      kubectl apply -f - <<EOF
${templatefile("${path.module}/deploy/odigos/clickhouse-destination.yaml", {
  ec2_ip = "$EC2_IP"
})}
EOF
      
      # Restart odigos-gateway deployment to pick up new destination
      echo "Restarting odigos-gateway deployment..."
      kubectl rollout restart deployment/odigos-gateway -n odigos-system
      
      echo "Odigos ClickHouse destination deployment completed!"
    EOT
  }

  triggers = {
    cluster_endpoint = module.eks.cluster_endpoint
    clickhouse_destination = filesha256("${path.module}/deploy/odigos/clickhouse-destination.yaml")
    ec2_ip = try(data.terraform_remote_state.ec2.outputs.monitoring_instance_private_ip, "pending")
    force_reinstall = "force-clickhouse-reinstall-$(date +%s)"
  }
}

