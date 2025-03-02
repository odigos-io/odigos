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
  default     = "m6a.xlarge"
}

variable "test_scenario" {
  description = "Test scenario to differentiate state files"
  type        = string
}

variable "run_id" {
  description = "GitHub run ID for uniquely identifying state files"
  type        = string
}
