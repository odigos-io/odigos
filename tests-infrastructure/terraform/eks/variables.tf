variable "cluster_name" {
  description = "Name of the EKS cluster"
  default     = "tests-eks"
}

variable "region" {
  description = "AWS region to deploy the cluster in"
  default     = "us-east-1"
}

variable "node_count" {
  description = "Number of nodes in the cluster"
  default     = 2
}

variable "node_spec" {
  description = "The node spec for the cluster"
  type        = string
  default     = null
}

variable "platform" {
  description = "Target CPU architecture for worker nodes"
  type        = string
  default     = "amd" # allowed: amd | arm

  validation {
    condition     = contains(["amd", "arm"], var.platform)
    error_message = "platform must be either \"amd\" or \"arm\"."
  }
}
