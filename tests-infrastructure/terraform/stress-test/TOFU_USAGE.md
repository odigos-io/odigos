# OpenTofu Usage Guide

This guide covers OpenTofu-specific commands and workflows for the Odigos stress testing infrastructure.

## Prerequisites

### Install OpenTofu
```bash
# macOS (using Homebrew)
brew install opentofu

# Linux (using package manager)
# Ubuntu/Debian
wget https://github.com/opentofu/opentofu/releases/download/v1.6.0/tofu_1.6.0_linux_amd64.deb
sudo dpkg -i tofu_1.6.0_linux_amd64.deb

# Verify installation
tofu version
```

### AWS Configuration
```bash
# Configure AWS credentials
aws configure

# Verify access
aws sts get-caller-identity
```

## Basic Commands

### Initialize Terraform
```bash
# Initialize main infrastructure
tofu init

# Initialize EC2 stack (if deploying separately)
cd ec2
tofu init
cd ..
```

### Plan and Apply
```bash
# Plan deployment
tofu plan

# Apply with auto-approval
tofu apply -auto-approve

# Apply with specific variables
tofu apply -var="deploy_kubernetes_apps=true" -auto-approve
```

### View Outputs
```bash
# View all outputs
tofu output

# View specific output
tofu output cluster_info
tofu output clickhouse_connection_info
tofu output ec2_instance_id

# View output in JSON format
tofu output -json
```

## Deployment Workflows

### Complete Deployment
```bash
# Deploy everything at once
./deploy.sh deploy
```

### Phased Deployment
```bash
# Phase 1: EKS infrastructure only
tofu apply -var="deploy_kubernetes_apps=false" -auto-approve

# Phase 2: EC2 monitoring stack
cd ec2
tofu init
tofu apply -auto-approve
cd ..

# Phase 3: Kubernetes applications
tofu apply -var="deploy_kubernetes_apps=true" -auto-approve
```

### Update Deployment
```bash
# Update with new configuration
tofu plan
tofu apply -auto-approve

# Update specific resources
tofu apply -target=module.eks -auto-approve
tofu apply -target=module.ec2 -auto-approve
```

## State Management

### View State
```bash
# List all resources
tofu state list

# Show specific resource
tofu state show module.eks.aws_eks_cluster.this[0]

# Show resource details
tofu show
```

### State Operations
```bash
# Import existing resource
tofu import aws_instance.example i-1234567890abcdef0

# Remove resource from state
tofu state rm aws_instance.example

# Move resource
tofu state mv aws_instance.old aws_instance.new
```

### State Backup
```bash
# Backup state file
cp terraform.tfstate terraform.tfstate.backup

# Backup with timestamp
cp terraform.tfstate "terraform.tfstate.backup.$(date +%Y%m%d_%H%M%S)"
```

## Variable Management

### Variable Files
```bash
# Use specific variable file
tofu apply -var-file="production.tfvars"

# Use multiple variable files
tofu apply -var-file="common.tfvars" -var-file="production.tfvars"

# Override variables
tofu apply -var="cluster_name=my-cluster" -var="node_count=5"
```

### Environment Variables
```bash
# Set Terraform variables
export TF_VAR_cluster_name="my-cluster"
export TF_VAR_region="us-west-2"
export TF_VAR_node_count="3"

# Apply with environment variables
tofu apply -auto-approve
```

### Variable Validation
```bash
# Validate configuration
tofu validate

# Format configuration
tofu fmt

# Check formatting
tofu fmt -check
```

## Resource Targeting

### Target Specific Resources
```bash
# Target specific module
tofu apply -target=module.eks -auto-approve

# Target specific resource
tofu apply -target=aws_eks_cluster.this -auto-approve

# Target multiple resources
tofu apply -target=module.eks -target=module.vpc -auto-approve
```

### Target by Type
```bash
# Target all EKS resources
tofu apply -target='module.eks.*' -auto-approve

# Target all EC2 resources
tofu apply -target='module.ec2.*' -auto-approve
```

## Workspace Management

### Create and Use Workspaces
```bash
# List workspaces
tofu workspace list

# Create new workspace
tofu workspace new production

# Switch workspace
tofu workspace select production

# Delete workspace
tofu workspace delete production
```

### Workspace-specific Variables
```bash
# Use workspace-specific variable file
tofu apply -var-file="workspaces/production.tfvars"

# Use workspace in variable file
# terraform.tfvars
cluster_name = "odigos-${terraform.workspace}"
```

## Debugging and Troubleshooting

### Debug Output
```bash
# Enable debug logging
export TF_LOG=DEBUG
tofu apply -auto-approve

# Enable trace logging
export TF_LOG=TRACE
tofu apply -auto-approve

# Disable logging
unset TF_LOG
```

### Plan Analysis
```bash
# Show detailed plan
tofu plan -detailed-exitcode

# Show plan in JSON format
tofu plan -out=plan.json
tofu show -json plan.json

# Show plan for specific resource
tofu plan -target=aws_eks_cluster.this
```

### Resource Inspection
```bash
# Show resource configuration
tofu show

# Show specific resource
tofu show aws_eks_cluster.this

# Show resource in JSON format
tofu show -json aws_eks_cluster.this
```

## Advanced Features

### Parallel Execution
```bash
# Control parallelism
tofu apply -parallelism=10 -auto-approve

# Disable parallelism
tofu apply -parallelism=1 -auto-approve
```

### Refresh State
```bash
# Refresh state from remote
tofu refresh

# Refresh specific resource
tofu refresh -target=aws_eks_cluster.this
```

### Import Resources
```bash
# Import existing EKS cluster
tofu import aws_eks_cluster.this odigos-stress-test

# Import existing VPC
tofu import module.vpc.aws_vpc.this[0] vpc-12345678
```

## Cleanup Commands

### Destroy Resources
```bash
# Destroy everything
tofu destroy -auto-approve

# Destroy specific resources
tofu destroy -target=module.ec2 -auto-approve

# Destroy with confirmation
tofu destroy
```

### Selective Cleanup
```bash
# Remove only Kubernetes applications
tofu apply -var="deploy_kubernetes_apps=false" -auto-approve

# Remove only EC2 stack
cd ec2
tofu destroy -auto-approve
cd ..
```

## Best Practices

### Configuration Management
- Use version control for all configuration files
- Keep sensitive data in separate variable files
- Use consistent naming conventions
- Document all custom variables

### State Management
- Store state in remote backend (S3 + DynamoDB)
- Enable state locking
- Regular state backups
- Use workspaces for different environments

### Security
- Use IAM roles with minimal permissions
- Encrypt sensitive data
- Rotate credentials regularly
- Monitor access logs

### Performance
- Use appropriate parallelism settings
- Target specific resources when possible
- Use data sources instead of hardcoded values
- Regular configuration optimization

## Common Issues and Solutions

### State Lock Issues
```bash
# Force unlock (use with caution)
tofu force-unlock <lock-id>

# Check lock status
tofu plan
```

### Provider Issues
```bash
# Update provider versions
tofu init -upgrade

# Clean provider cache
rm -rf .terraform/providers/
tofu init
```

### Resource Conflicts
```bash
# Check for conflicts
tofu plan

# Resolve conflicts manually
tofu import <resource_type>.<name> <resource_id>
```

### Variable Issues
```bash
# Check variable values
tofu console
> var.cluster_name

# Validate variables
tofu validate
```

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Deploy Infrastructure
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup OpenTofu
        uses: opentofu/setup-opentofu@v1
        with:
          tofu_version: 1.6.0
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v1
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1
      - name: Deploy
        run: |
          tofu init
          tofu plan
          tofu apply -auto-approve
```

### GitLab CI Example
```yaml
deploy:
  stage: deploy
  image: opentofu/opentofu:1.6.0
  before_script:
    - tofu init
  script:
    - tofu plan
    - tofu apply -auto-approve
  only:
    - main
```

## Monitoring and Alerting

### Terraform Cloud Integration
```bash
# Login to Terraform Cloud
tofu login

# Configure remote backend
# backend.tf
terraform {
  backend "remote" {
    organization = "your-org"
    workspaces {
      name = "odigos-stress-test"
    }
  }
}
```

### Custom Monitoring
```bash
# Check deployment status
tofu output -json | jq '.cluster_info.value'

# Monitor resource changes
tofu plan -out=plan.json
tofu show -json plan.json | jq '.resource_changes[]'
```

## Support and Resources

### Documentation
- [OpenTofu Documentation](https://opentofu.org/docs/)
- [AWS Provider Documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [EKS Module Documentation](https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/latest)

### Community
- [OpenTofu GitHub](https://github.com/opentofu/opentofu)
- [Terraform Community](https://discuss.hashicorp.com/c/terraform-core)
- [AWS EKS Community](https://github.com/aws/containers-roadmap)

### Troubleshooting
- Check OpenTofu logs with `TF_LOG=DEBUG`
- Review AWS CloudTrail for API errors
- Check EKS cluster logs in CloudWatch
- Verify IAM permissions and policies