# Odigos Stress Testing Infrastructure

Complete infrastructure setup for running stress tests on Odigos with integrated monitoring and telemetry collection.

## Deployment Options

### Quick Start (Automated)
```bash
# Deploy everything:  EKS cluster, EC2 monitoring stack, and all Kubernetes applications including load-test workloads.
./deploy.sh deploy

# Check status
./deploy.sh status

# Deploy only EKS applications
./deploy.sh k8s-apps
```

### Kubernetes Applications Only
```bash
# Deploy core applications (Odigos, Prometheus) without load test workloads
./deploy.sh k8s-apps

# Deploy core applications with load test workloads (span generators)
./deploy.sh k8s-apps --with-load-test
```

```

### Configuration

Edit `terraform.tfvars` to customize settings:

```hcl
cluster_name = "your-cluster-name"
region       = "us-east-1"
node_spec    = "c6a.2xlarge"
node_desired_size = 3
node_max_size = 5
```

### Data Flow
```
Test Apps (Odigos Sources) → Data Collection → Odigos Gateway → ClickHouse (EC2)
              ↓                    ↓              ↓
         Prometheus (EC2) ←────────┴──────────────┘
              ↓
          Grafana (EC2)
```

## Workload Generators

The infrastructure includes two types of span generators for load testing:

### Continuous Generators (`generators/`)
- **Purpose**: Long-running span generation for sustained load testing
- **Languages**: Go, Python, Node.js, Java
- **Behavior**: Continuously generate spans until manually stopped
- **Use Case**: Endurance testing and monitoring system behavior under sustained load

### Fixed Span Generators (`fixed-span-generators/`)
- **Purpose**: Controlled, predictable load testing scenarios
- **Languages**: Go, Python, Node.js, Java
- **Behavior**: Generate exactly 10,000 spans per run, then terminate
- **Use Case**: Performance benchmarking, burst testing, and controlled experiments

## Directory Structure

```
stress-test/
├── README.md                    # This file
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
│   │   ├── generators/          # Continuous span generators (Go, Java, Node, Python)
│   │   ├── fixed-span-generators/ # Fixed span generators (10,000 spans each)
│   │   └── applications/        # Full applications
│   ├── monitoring-stack/        # Prometheus configuration
│   └── odigos/                  # Odigos configurations
├── deploy.sh                    # Main deployment script
└── TOFU_USAGE.md               # Tofu-specific instructions
```

## Monitoring Access

Services run on EC2 instance. Use AWS SSM for port forwarding:

```bash
# Get instance ID
cd ec2/ && tofu output monitoring_instance_id && cd ..

# Access services (replace <instance-id>)
aws ssm start-session --target <instance-id> --document-name AWS-StartPortForwardingSession --parameters '{"portNumber":["3000"],"localPortNumber":["3000"]}'  # Grafana
aws ssm start-session --target <instance-id> --document-name AWS-StartPortForwardingSession --parameters '{"portNumber":["9090"],"localPortNumber":["9090"]}'  # Prometheus
aws ssm start-session --target <instance-id> --document-name AWS-StartPortForwardingSession --parameters '{"portNumber":["8123"],"localPortNumber":["8123"]}'  # ClickHouse
```

## Cleanup

```bash
# Automated cleanup
./deploy.sh destroy
