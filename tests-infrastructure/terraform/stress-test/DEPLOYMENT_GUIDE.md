# Odigos Stress Testing Infrastructure - Deployment Guide

This guide will walk you through deploying the complete stress testing infrastructure for Odigos.

## Prerequisites

### Required Tools

1. **AWS CLI** - Configured with appropriate permissions
2. **Terraform/OpenTofu** >= 1.0.0
3. **kubectl** - For Kubernetes cluster interaction
4. **Docker** - For building test applications

### AWS Permissions

Your AWS credentials need the following permissions:
- EC2 (VPC, Security Groups, Instances, EBS)
- EKS (Clusters, Node Groups, IAM Roles)
- IAM (Role creation and attachment)
- Systems Manager (for EC2 access)

## Quick Start

### Automated Deployment

```bash
# Deploy everything automatically
./scripts/deploy.sh

# Check status
./scripts/deploy.sh status

# Deploy only applications
./scripts/deploy.sh k8s-apps
```

### Manual Deployment

#### Step 1: Configure Infrastructure

1. **Edit terraform.tfvars:**
   ```hcl
   # Basic Configuration
   cluster_name = "your-odigos-stress-test"
   region       = "us-east-1"
   
   # Security - IMPORTANT: Restrict this in production!
   cluster_endpoint_public_access_cidrs = ["YOUR_IP/32"]
   
   # Node Configuration
   node_spec        = "c6a.2xlarge"
   node_desired_size = 3
   node_max_size    = 5
   ```

#### Step 2: Deploy Infrastructure

```bash
# Deploy EKS infrastructure
tofu init
tofu plan
tofu apply

# Deploy monitoring stack
cd ec2/
tofu init
tofu apply
cd ..

# Connect monitoring to EKS
tofu apply
```

#### Step 3: Deploy Applications

```bash
# Configure kubectl
aws eks update-kubeconfig --region us-east-1 --name your-cluster-name

# Deploy test workloads
kubectl apply -f deploy/workloads/ --recursive

# Deploy Odigos configurations
kubectl apply -f deploy/odigos/
```

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   EKS Cluster   │    │  Monitoring      │    │  Test Workloads │
│                 │    │  Stack           │    │                 │
│ • Kubernetes    │◄──►│ • Prometheus     │    │ • Span Gen      │
│ • Node Groups   │    │ • Grafana        │    │ • HTTP Servers  │
│ • Applications  │    │ • ClickHouse     │    │ • Go Apps       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Directory Structure

```
stress-test/
├── README.md                    # Quick start guide
├── main.tf                      # EKS infrastructure
├── variables.tf                 # EKS variables
├── outputs.tf                   # EKS outputs
├── provider.tf                  # AWS provider config
├── terraform.tfvars             # Configuration
├── ec2/                         # Monitoring stack
│   ├── main.tf                  # EC2 instance with monitoring tools
│   ├── variables.tf             # EC2 variables
│   └── outputs.tf               # EC2 outputs
├── deploy/                      # Kubernetes manifests
│   ├── workloads/               # Test applications
│   │   ├── generators/          # Span generators (Go, Java, Node, Python)
│   │   ├── services/            # HTTP services
│   │   └── applications/        # Full applications
│   ├── monitoring-stack/        # Prometheus configuration
│   └── odigos/                  # Odigos configurations
├── scripts/                     # Deployment scripts
│   └── deploy.sh                # Main deployment script
└── docs/                        # Documentation
    ├── DEPLOYMENT_GUIDE.md      # This file
    └── README.md                # Architecture overview
```

## Monitoring Stack

The monitoring EC2 instance includes:

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards  
- **ClickHouse**: High-performance data storage
- **K6**: Load testing framework

### Accessing Services

All monitoring services run on the EC2 instance. Use AWS Systems Manager Session Manager for port forwarding:

```bash
# Get monitoring instance ID
cd ec2/
tofu output monitoring_instance_private_ip
cd ..

# Grafana (port 3000)
aws ssm start-session --target <instance-id> \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["3000"],"localPortNumber":["3000"]}'

# Prometheus (port 9090)
aws ssm start-session --target <instance-id> \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["9090"],"localPortNumber":["9090"]}'

# ClickHouse HTTP (port 8123)
aws ssm start-session --target <instance-id> \
  --document-name AWS-StartPortForwardingSession \
  --parameters '{"portNumber":["8123"],"localPortNumber":["8123"]}'
```

## Test Applications

### Span Generators

Multi-language applications that generate telemetry data:

- **Go**: High-performance span generation
- **Java**: Spring Boot application
- **Node.js**: Express.js application  
- **Python**: FastAPI application

### Deployment

```bash
# Deploy all span generators
kubectl apply -f deploy/workloads/generators/

# Deploy specific language
kubectl apply -f deploy/workloads/generators/go/
kubectl apply -f deploy/workloads/generators/java/
kubectl apply -f deploy/workloads/generators/node/
kubectl apply -f deploy/workloads/generators/python/
```

## Odigos Integration

### ClickHouse Destination

The infrastructure automatically configures Odigos to send telemetry data to ClickHouse:

```bash
# Check destination status
kubectl get destinations -n odigos-system

# Check destination details
kubectl describe destination clickhouse-destination -n odigos-system
```

### Data Flow

```
EKS Applications → Odigos Collector → ClickHouse (EC2)
                                      ↓
                                 Grafana (EC2)
```

## Running Load Tests

### Using K6

The K6 load testing script is pre-configured on the monitoring instance:

```bash
# SSH to monitoring instance
aws ssm start-session --target <instance-id>

# Run load test
sudo systemctl start k6-loadtest

# Check status
sudo systemctl status k6-loadtest

# View logs
sudo journalctl -u k6-loadtest -f
```

## Verification Checklist

- [ ] EKS cluster is running and accessible
- [ ] All nodes are in Ready state
- [ ] Test applications are deployed and running
- [ ] Monitoring services are accessible
- [ ] Odigos destination is configured
- [ ] Metrics are flowing to Prometheus
- [ ] Telemetry data is flowing to ClickHouse
- [ ] Grafana dashboards are populated

## Troubleshooting

### Common Issues

1. **Cluster not accessible:**
   ```bash
   # Check security groups
   aws ec2 describe-security-groups --group-ids <sg-id>
   
   # Verify kubectl config
   kubectl config current-context
   ```

2. **Applications not starting:**
   ```bash
   # Check pod status
   kubectl get pods --all-namespaces
   
   # Check pod logs
   kubectl logs <pod-name> -n <namespace>
   
   # Check resource limits
   kubectl describe pod <pod-name> -n <namespace>
   ```

3. **Monitoring not accessible:**
   ```bash
   # Check EC2 instance status
   aws ec2 describe-instances --instance-ids <instance-id>
   
   # Check security groups
   aws ec2 describe-security-groups --group-ids <sg-id>
   
   # Check SSM agent
   aws ssm describe-instance-information --filters "Key=InstanceIds,Values=<instance-id>"
   ```

4. **Odigos destination issues:**
   ```bash
   # Check destination conditions
   kubectl describe destination clickhouse-destination -n odigos-system
   
   # Check ClickHouse connectivity
   kubectl run test-clickhouse --image=busybox --rm -it --restart=Never -- sh -c "nc -zv <EC2_IP> 9000"
   ```

### Useful Commands

```bash
# Check cluster status
kubectl get nodes
kubectl get pods --all-namespaces

# Check resource usage
kubectl top nodes
kubectl top pods --all-namespaces

# Check services
kubectl get svc --all-namespaces

# View terraform outputs
tofu output

# Check EC2 status
cd ec2/
tofu output
```

## Cleanup

To destroy the infrastructure:

```bash
# Automated cleanup
./scripts/deploy.sh destroy

# Manual cleanup
kubectl delete -f deploy/ --recursive
tofu destroy
cd ec2/
tofu destroy
```

## Security Best Practices

1. **Network Security:**
   - Use private subnets for all resources
   - Restrict EKS endpoint access to specific IPs
   - Use security groups with least privilege

2. **Access Control:**
   - Use IAM roles instead of access keys
   - Enable MFA for AWS console access
   - Use AWS Systems Manager for EC2 access

3. **Data Protection:**
   - Enable encryption for all EBS volumes
   - Use AWS Secrets Manager for sensitive data
   - Enable CloudTrail for audit logging

4. **Monitoring:**
   - Set up CloudWatch alarms
   - Enable VPC Flow Logs
   - Monitor resource usage and costs

## Cost Optimization Tips

1. **Use Spot Instances:** Configure node groups to use spot instances for non-critical workloads
2. **Auto Scaling:** Set up HPA for test applications
3. **Resource Limits:** Set appropriate CPU/memory limits
4. **Scheduled Scaling:** Use KEDA or similar tools for scheduled scaling
5. **Cleanup:** Destroy infrastructure when not in use
6. **Monitoring:** Set up cost alerts and budgets

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review AWS and Kubernetes documentation
3. Check the main README.md file
4. Contact the platform team