# =============================================================================
# CLUSTER CONFIGURATION
# =============================================================================

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "odigos-stress-test"
}

variable "region" {
  description = "AWS region to deploy the cluster in"
  type        = string
  default     = "us-east-1"
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.32"
}

variable "deploy_kubernetes_apps" {
  description = "Whether to deploy Kubernetes applications (set to false for infrastructure-only deployment)"
  type        = bool
  default     = true
}

variable "deploy_load_test_apps" {
  description = "Whether to deploy load-test applications (workload generators)"
  type        = bool
  default     = false
}

# =============================================================================
# NETWORK CONFIGURATION
# =============================================================================

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones_count" {
  description = "Number of availability zones to use"
  type        = number
  default     = 2
  validation {
    condition     = var.availability_zones_count >= 2
    error_message = "At least 2 availability zones are required."
  }
}

variable "private_subnets" {
  description = "CIDR blocks for private subnets"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

variable "public_subnets" {
  description = "CIDR blocks for public subnets"
  type        = list(string)
  default     = ["10.0.101.0/24", "10.0.102.0/24"]
}

variable "single_nat_gateway" {
  description = "Use a single NAT Gateway for all private subnets"
  type        = bool
  default     = true
}

# =============================================================================
# EKS CONFIGURATION
# =============================================================================

variable "cluster_endpoint_private_access" {
  description = "Enable private access to the EKS cluster"
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access" {
  description = "Enable public access to the EKS cluster"
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access_cidrs" {
  description = "List of CIDR blocks that can access the EKS cluster publicly"
  type        = list(string)
  default     = ["0.0.0.0/0"] # Restrict this in production!
}

variable "node_ami_type" {
  description = "AMI type for the EKS managed node groups"
  type        = string
  default     = "AL2_x86_64"
}

variable "node_spec" {
  description = "Instance type for the EKS nodes"
  type        = string
  default     = "c6a.2xlarge"
}

variable "node_min_size" {
  description = "Minimum number of nodes in the EKS node group"
  type        = number
  default     = 1
}

variable "node_desired_size" {
  description = "Desired number of nodes in the EKS node group"
  type        = number
  default     = 3
}

variable "node_max_size" {
  description = "Maximum number of nodes in the EKS node group"
  type        = number
  default     = 5
}

variable "node_disk_size" {
  description = "Disk size for EKS nodes in GB"
  type        = number
  default     = 50
}

# =============================================================================
# SECURITY CONFIGURATION
# =============================================================================

variable "enable_vpc_endpoint" {
  description = "Enable VPC endpoint for EKS API"
  type        = bool
  default     = false
}

variable "create_monitoring_connection" {
  description = "Create security group rule to connect monitoring EC2 to EKS"
  type        = bool
  default     = false
}

# =============================================================================
# ODIGOS CONFIGURATION
# =============================================================================

variable "odigos_tag" {
  description = "Odigos image tag to deploy"
  type        = string
  default     = "latest"
}

variable "odigos_api_key" {
  description = "Odigos API key for on-premises deployment"
  type        = string
  default     = ""
  sensitive   = true
}
