# Odigos Stress Test Infrastructure - Deployment Guide

This guide provides detailed instructions for deploying and managing the Odigos stress testing infrastructure.

## Overview

The infrastructure consists of:
- **EKS Cluster**: Kubernetes cluster with Odigos for automatic telemetry collection
- **EC2 Monitoring Stack**: Prometheus, Grafana, ClickHouse, and K6 for monitoring and testing
- **Test Workloads**: Span generators and test applications for load testing

## Prerequisites

### Required Tools
- **AWS CLI**: Configured with appropriate permissions
- **OpenTofu/Terraform**: >= 1.0.0
- **kubectl**: For Kubernetes cluster management
- **Docker**: For building test applications (optional)

### AWS Permissions
Your AWS credentials need permissions for:
- EKS cluster creation and management
- EC2 instance creation and management
- VPC and networking configuration
- IAM role and policy management
- Systems Manager (for EC2 access)

## Quick Deployment

### 1. Clone and Navigate
```bash
git clone <repository>
cd tests-infrastructure/terraform/stress-test
```

### 2. Deploy Everything
```bash
# Deploy complete infrastructure (EKS + EC2 + Kubernetes apps)
./deploy.sh deploy

# Check deployment status
./deploy.sh status
```

### 3. Verify Deployment
```bash
# Check cluster status
kubectl get nodes

# Check Odigos status
kubectl get pods -n odigos-system

# Check workload generators
kubectl get pods -n load-test

# Check destinations
kubectl get destinations -n odigos-system
```

## Detailed Deployment Process

### Phase 1: EKS Infrastructure
```bash
# Deploy EKS cluster and networking
tofu apply -var="deploy_kubernetes_apps=false" -auto-approve
```

This creates:
- VPC with public/private subnets
- EKS cluster with managed node groups
- Security groups and IAM roles
- KMS encryption keys

### Phase 2: EC2 Monitoring Stack
```bash
# Deploy EC2 monitoring instance
cd ec2
tofu init
tofu apply -auto-approve
cd ..
```

This creates:
- EC2 instance with monitoring stack
- Prometheus, Grafana, ClickHouse, K6
- Encrypted EBS volumes
- Security groups for EKS communication

### Phase 3: Kubernetes Applications
```bash
# Deploy Kubernetes applications
tofu apply -var="deploy_kubernetes_apps=true" -auto-approve
```

This deploys:
- Prometheus agent in EKS
- Odigos for telemetry collection
- Workload generators (Go, Java, Node.js, Python)
- Odigos sources and ClickHouse destination

## Configuration

### Terraform Variables
Edit `terraform.tfvars` to customize your deployment:

```hcl
# Cluster configuration
cluster_name = "odigos-stress-test"
region       = "us-east-1"
node_count   = 3
node_spec    = "c6a.2xlarge"

# EC2 configuration
ec2_instance_type = "m6i.large"
ec2_volume_size  = 100

# Monitoring configuration
prometheus_retention = "7d"
grafana_admin_password = "admin"
clickhouse_password = "stresstest"
```

### Workload Generator Configuration
Workload generators are configured in `deploy/workloads/generators/`:
- **Go**: High-performance span generation
- **Java**: JVM-based workload testing
- **Node.js**: JavaScript/TypeScript applications
- **Python**: Python-based applications

Each generator includes:
- Pre-configured span generation
- Odigos auto-instrumentation labels
- Resource limits and scaling

## Accessing Services

### Get Connection Information
```bash
# Get all outputs
tofu output

# Get specific information
tofu output cluster_info
tofu output clickhouse_connection_info
tofu output ec2_instance_id
```

### Port Forwarding to EC2 Services
```bash
# Get instance ID
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

### Service URLs
After port forwarding:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **ClickHouse HTTP**: http://localhost:8123
- **ClickHouse Native**: tcp://<EC2_IP>:9000

## Monitoring and Observability

### Odigos Telemetry Collection
Odigos automatically:
- Detects applications with `odigos-target=true` label
- Instruments them for traces, metrics, and logs
- Routes telemetry to ClickHouse destination

### Data Flow
```
Test Applications → Odigos Sources → Odigos Gateway → ClickHouse (EC2)
                                                      ↓
                                                 Prometheus (EC2)
                                                      ↓
                                                 Grafana (EC2)
```

### ClickHouse Tables
The following tables are automatically created:
- `otel_traces` - OpenTelemetry traces
- `otel_logs` - Application logs
- `otel_metrics_gauge` - Gauge metrics
- `otel_metrics_sum` - Sum metrics
- `otel_metrics_histogram` - Histogram metrics
- `otel_metrics_exponential_histogram` - Exponential histogram metrics
- `otel_metrics_summary` - Summary metrics

## Load Testing

### K6 Integration
K6 is pre-installed on the EC2 instance for load testing:

```bash
# Access EC2 instance
aws ssm start-session --target $INSTANCE_ID

# Check K6 status
sudo systemctl status k6-loadtest

# View K6 logs
sudo journalctl -u k6-loadtest -f

# Run custom K6 tests
sudo nano /opt/k6/tests/loadtest.js
sudo systemctl restart k6-loadtest
```

### Test Scenarios
Default K6 script includes:
- Health check endpoints
- API status checks
- POST request testing
- Configurable via environment variables

## Troubleshooting

### Common Issues

#### 1. EKS Cluster Not Ready
```bash
# Check cluster status
aws eks describe-cluster --name odigos-stress-test

# Check node groups
aws eks describe-nodegroup --cluster-name odigos-stress-test --nodegroup-name stress-test-nodes

# Check node status
kubectl get nodes
```

#### 2. Odigos Not Working
```bash
# Check Odigos pods
kubectl get pods -n odigos-system

# Check destination status
kubectl describe destination clickhouse-destination -n odigos-system

# Check gateway logs
kubectl logs -l app.kubernetes.io/name=odigos -n odigos-system

# Check sources
kubectl get sources -n load-test
```

#### 3. ClickHouse Connection Issues
```bash
# Test connectivity from EKS
kubectl run test-clickhouse --image=busybox --rm -it --restart=Never -- sh -c "nc -zv <EC2_IP> 9000"

# Check ClickHouse on EC2
aws ssm start-session --target $INSTANCE_ID
sudo systemctl status clickhouse-server
ss -tlnp | grep 9000
```

#### 4. EC2 Services Not Starting
```bash
# Check service status
sudo systemctl status prometheus grafana-server clickhouse-server

# View service logs
sudo journalctl -u prometheus -f
sudo journalctl -u grafana-server -f
sudo journalctl -u clickhouse-server -f

# Check disk space
df -h
```

### Useful Commands

#### EKS Cluster
```bash
# Update kubeconfig
aws eks update-kubeconfig --region us-east-1 --name odigos-stress-test

# Check all resources
kubectl get all --all-namespaces

# Check specific namespaces
kubectl get pods -n odigos-system
kubectl get pods -n load-test
kubectl get pods -n monitoring
```

#### EC2 Instance
```bash
# Get instance ID
INSTANCE_ID=$(tofu output -raw ec2_instance_id)

# Check instance status
aws ec2 describe-instances --instance-ids $INSTANCE_ID

# Access instance
aws ssm start-session --target $INSTANCE_ID

# Check services
sudo systemctl status prometheus grafana-server clickhouse-server k6-loadtest

# Check disk usage
df -h
lsblk
```

## Scaling and Performance

### EKS Cluster Scaling
```bash
# Scale node group
aws eks update-nodegroup-config \
  --cluster-name odigos-stress-test \
  --nodegroup-name stress-test-nodes \
  --scaling-config minSize=2,maxSize=10,desiredSize=5
```

### Workload Generator Scaling
```bash
# Scale specific generators
kubectl scale deployment go-span-generator -n load-test --replicas=5
kubectl scale deployment java-span-generator -n load-test --replicas=3
```

### EC2 Instance Scaling
For high-load scenarios, consider:
- Upgrading EC2 instance type
- Increasing EBS volume sizes
- Adding additional monitoring instances

## Security Considerations

### Network Security
- EKS cluster runs in private subnets
- EC2 instance accessible only via Systems Manager
- Security groups configured for least privilege access
- All EBS volumes encrypted

### Access Control
- IAM roles with minimal required permissions
- No SSH keys required (Systems Manager only)
- KMS encryption for sensitive data

### Data Protection
- All telemetry data encrypted in transit
- ClickHouse data encrypted at rest
- Prometheus metrics encrypted at rest

## Cleanup

### Complete Cleanup
```bash
# Destroy everything
./deploy.sh destroy
```

### Partial Cleanup
```bash
# Destroy only Kubernetes applications
tofu apply -var="deploy_kubernetes_apps=false" -auto-approve

# Destroy only EC2 stack
cd ec2
tofu destroy
cd ..

# Destroy only EKS cluster
tofu destroy
```

### Manual Cleanup
```bash
# Delete EKS cluster
aws eks delete-cluster --name odigos-stress-test

# Delete EC2 instance
aws ec2 terminate-instances --instance-ids <instance-id>

# Clean up Terraform state
rm -rf .terraform/
rm terraform.tfstate*
```

## Support and Maintenance

### Regular Maintenance
- Monitor disk usage on EC2 instance
- Check service health and logs
- Update software versions as needed
- Review and rotate credentials

### Monitoring
- Set up CloudWatch alarms for critical metrics
- Monitor EKS cluster health
- Track EC2 instance performance
- Monitor ClickHouse storage usage

### Backup
- EBS snapshots for EC2 volumes
- EKS cluster configuration backup
- Terraform state backup
- ClickHouse data export (if needed)

## Next Steps

After successful deployment:
1. **Access Grafana**: Set up dashboards for your specific use case
2. **Configure Alerts**: Set up Prometheus alerts for critical metrics
3. **Customize Workloads**: Modify test applications for your specific scenarios
4. **Scale Testing**: Run load tests with K6 to validate performance
5. **Monitor Data Flow**: Verify telemetry data is flowing correctly through the pipeline

For additional support or questions, refer to the main README.md or create an issue in the repository.