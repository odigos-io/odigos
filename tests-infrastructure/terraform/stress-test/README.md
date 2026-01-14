# Odigos Stress Testing Infrastructure

This directory contains the complete infrastructure setup for running stress tests on Odigos with integrated monitoring and telemetry collection.

## Quick Start

```bash
# Deploy everything (EKS + EC2 + Kubernetes apps + load-test workloads)
./deploy.sh deploy

# Deploy only Kubernetes applications (without load test workloads)
./deploy.sh k8s-apps

# Deploy Kubernetes applications with load test workloads
./deploy.sh k8s-apps --with-load-test

# Check status
./deploy.sh status

# Get cluster info
tofu output cluster_info

# Get ClickHouse connection info
tofu output clickhouse_connection_info
```

## Directory Structure

```
stress-test/
├── main.tf                 # EKS cluster configuration
├── ec2/                    # Monitoring stack (EC2)
├── deploy/                 # Kubernetes manifests
│   ├── workloads/          # Test applications
│   ├── monitoring-stack/   # Prometheus, Grafana
│   └── odigos/            # Odigos configurations
├── deploy.sh              # Main deployment script
├── README.md              # This file
├── DEPLOYMENT_GUIDE.md    # Detailed deployment guide
└── TOFU_USAGE.md         # Tofu-specific instructions
```

## Deployment Options

### Full Deployment
```bash
./deploy.sh deploy
```
Deploys everything: EKS cluster, EC2 monitoring stack, and all Kubernetes applications including load-test workloads.

### Kubernetes Applications Only
```bash
# Deploy core applications (Odigos, Prometheus) without load test workloads
./deploy.sh k8s-apps

# Deploy core applications with load test workloads (span generators)
./deploy.sh k8s-apps --with-load-test
```

### Infrastructure Only
```bash
# Deploy only EKS cluster
./deploy.sh infrastructure

# Deploy only EC2 monitoring stack
./deploy.sh ec2
```

## Documentation

- [Deployment Guide](DEPLOYMENT_GUIDE.md)
- [Tofu Usage](TOFU_USAGE.md)

## Accessing Services

### Get Connection Information
```bash
# Get ClickHouse connection info
tofu output clickhouse_connection_info

# Get EC2 instance ID
tofu output ec2_instance_id

# Get cluster info
tofu output cluster_info
```

### Port Forwarding to EC2 Services
```bash
# Get instance ID first
INSTANCE_ID=$(tofu output -raw ec2_instance_id)

# Grafana (port 3000)
aws ssm start-session --target $INSTANCE_ID \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["3000"],"localPortNumber":["3000"]}'

# Prometheus (port 9090)
aws ssm start-session --target $INSTANCE_ID \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["9090"],"localPortNumber":["9090"]}'

# ClickHouse HTTP (port 8123)
aws ssm start-session --target $INSTANCE_ID \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["8123"],"localPortNumber":["8123"]}'
```

## Key Features

- **EKS Cluster**: Managed Kubernetes cluster with auto-scaling
- **Monitoring Stack**: Prometheus, Grafana, ClickHouse on EC2
- **Odigos Integration**: Automatic telemetry collection and routing
- **Test Workloads**: High-performance span generators (Go, Java, Node.js, Python)
- **Automatic Instrumentation**: Odigos auto-detects and instruments applications
- **Load Testing**: K6 integration for performance testing


## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   EKS Cluster   │    │  EC2 Monitoring  │    │  Test Workloads │
│                 │    │  Stack           │    │                 │
│ • Kubernetes    │◄──►│ • Prometheus     │    │ • Span Gen      │
│ • Odigos        │    │ • Grafana        │    │ • Test Apps     │
│ • Applications  │    │ • ClickHouse     │    │ • Auto-instr.   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Data Flow
```
Test Apps → Odigos Sources → Odigos Gateway → ClickHouse (EC2)
                                              ↓
                                         Prometheus (EC2)
                                              ↓
                                          Grafana (EC2)
```

## Prerequisites

- AWS CLI configured
- Terraform/OpenTofu >= 1.0.0
- kubectl
- Docker (for building test applications)

## Configuration

Edit `terraform.tfvars` to customize your deployment:

```hcl
cluster_name = "your-stress-test-cluster"
region       = "us-east-1"
node_count   = 3
node_spec    = "c6a.2xlarge"
```

## Monitoring

### Service URLs (after port forwarding)
- **Grafana**: http://localhost:3000 (admin/admin)
  - Kubernetes Pods View dashboard (ID: 15760) - Comprehensive pod monitoring
- **Prometheus**: http://localhost:9090
- **ClickHouse HTTP**: http://localhost:8123
- **ClickHouse Native**: tcp://<EC2_IP>:9000

### Odigos Telemetry
- **Odigos UI**: Available in EKS cluster
- **Sources**: Auto-detected workload generators
- **Destinations**: ClickHouse integration configured
- **Data Flow**: Traces, Metrics, Logs → ClickHouse → Grafana

## Troubleshooting

### EKS Cluster
```bash
# Check cluster status
kubectl get nodes
kubectl get pods --all-namespaces

# Check Odigos status
kubectl get pods -n odigos-system
kubectl get sources -n load-test
kubectl get destinations -n odigos-system

# Check workload generators
kubectl get pods -n load-test
```

### EC2 Monitoring Stack
```bash
# Get instance ID
INSTANCE_ID=$(tofu output -raw ec2_instance_id)

# Check instance status
aws ec2 describe-instances --instance-ids $INSTANCE_ID

# Check services via SSM
aws ssm start-session --target $INSTANCE_ID
sudo systemctl status prometheus grafana-server clickhouse-server
```

### Odigos Integration
```bash
# Check destination configuration
kubectl describe destination clickhouse-destination -n odigos-system

# Check gateway logs
kubectl logs -l app.kubernetes.io/name=odigos -n odigos-system

# Test ClickHouse connectivity
kubectl run test-clickhouse --image=busybox --rm -it --restart=Never -- sh -c "nc -zv <EC2_IP> 9000"
```

### View Terraform Outputs
```bash
tofu output
```

## Cleanup

```bash
# Destroy infrastructure
./deploy.sh destroy
```