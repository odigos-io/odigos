# EC2 Monitoring Stack

This directory contains the Terraform configuration for deploying the monitoring stack on EC2, including Prometheus, Grafana, ClickHouse, and K6. This stack is automatically deployed as part of the main infrastructure deployment.

## Architecture

The EC2 instance runs:
- **Prometheus**: Metrics collection and storage with remote write from EKS
- **Grafana**: Visualization and dashboards with pre-configured data sources
- **ClickHouse**: High-performance data storage for Odigos telemetry data
- **K6**: Load testing framework for performance testing

## Prerequisites

1. **EKS infrastructure must be deployed first** (main directory)
2. **terraform.tfstate** must exist in the parent directory
3. **AWS credentials** configured

## Quick Start

**Note**: This EC2 stack is automatically deployed as part of the main infrastructure. You typically don't need to deploy it separately.

### Manual Deployment (if needed)

1. **Configure the deployment:**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

2. **Deploy the monitoring stack:**
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. **Get the security group ID:**
   ```bash
   terraform output monitoring_ec2_sg_id
   ```

4. **Update main infrastructure:**
   ```bash
   cd ..
   # Add monitoring_sg_id to terraform.tfvars
   terraform apply
   ```

## Configuration

### Instance Configuration
- **Instance Type**: Configurable (default: m6i.large)
- **Volumes**: Separate encrypted volumes for each service
- **Security**: Systems Manager access only (no SSH)

### Software Versions
- **Prometheus**: 2.52.0 (configurable)
- **K6**: 0.51.0 (configurable)
- **Grafana**: Latest from official repository
- **ClickHouse**: Latest from official repository

### Security Features
- **Encrypted Volumes**: All EBS volumes are encrypted
- **Private Subnet**: Instance runs in private subnet
- **Systems Manager**: Access via SSM (no SSH keys needed)
- **Security Groups**: Least privilege access rules

## Accessing Services

### Port Forwarding Commands

```bash
# Get instance ID
INSTANCE_ID=$(terraform output -raw monitoring_instance_id)

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
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **ClickHouse**: http://localhost:8123

## Monitoring Configuration

### Prometheus
- **Remote Write**: Enabled for receiving metrics from EKS
- **Retention**: 7 days (configurable)
- **Storage**: Dedicated EBS volume

### Grafana
- **Data Source**: Pre-configured Prometheus connection
- **Storage**: Dedicated EBS volume
- **Authentication**: Default admin/admin (change in production)
- **Dashboards**: 
  - Kubernetes Pods View (ID: 15760) - Comprehensive pod monitoring

### ClickHouse
- **Storage**: Dedicated EBS volume
- **Authentication**: Default user with password `stresstest`
- **Ports**: 8123 (HTTP), 9000 (Native TCP)
- **Database**: `otel` (auto-created by Odigos)
- **Tables**: Auto-created for traces, metrics, and logs

## Load Testing with K6

### Configuration
The K6 load testing script is pre-configured with generic test scenarios and can be customized:

```bash
# SSH to instance via SSM
aws ssm start-session --target $INSTANCE_ID

# Edit the test script
sudo nano /opt/k6/tests/loadtest.js

# Configure target service URL
sudo systemctl edit k6-loadtest
# Add: Environment=K6_TARGET_SERVICE_URL=https://your-app.example.com

# Start load testing
sudo systemctl start k6-loadtest

# Check status
sudo systemctl status k6-loadtest

# View logs
sudo journalctl -u k6-loadtest -f
```

### Test Scenarios
The default script includes:
- **Health Check**: Tests `/health` endpoint
- **API Status**: Tests `/api/status` endpoint  
- **POST Request**: Tests `/api/data` endpoint with JSON payload

### Custom Tests
Create custom K6 scripts and place them in `/opt/k6/tests/` on the instance. The script uses environment variables for configuration.

## Troubleshooting

### Common Issues

1. **Instance not accessible via SSM**
   ```bash
   # Check instance status
   aws ec2 describe-instances --instance-ids $INSTANCE_ID
   
   # Check SSM agent
   aws ssm describe-instance-information --filters "Key=InstanceIds,Values=$INSTANCE_ID"
   ```

2. **Services not starting**
   ```bash
   # Check system logs
   sudo journalctl -u prometheus -f
   sudo journalctl -u grafana-server -f
   sudo journalctl -u clickhouse-server -f
   ```

3. **Volume mounting issues**
   ```bash
   # Check disk usage
   df -h
   
   # Check volume status
   lsblk
   ```

### Useful Commands

```bash
# Check service status
sudo systemctl status prometheus grafana-server clickhouse-server

# View service logs
sudo journalctl -u <service-name> -f

# Check disk usage
df -h

# Check network connectivity
curl -s http://localhost:9090/api/v1/status/config
curl -s http://localhost:3000/api/health
curl -s http://localhost:8123/ping
```


## Cleanup

To destroy the monitoring stack:

```bash
terraform destroy
```

