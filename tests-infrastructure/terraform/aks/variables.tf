variable "resource_group_name" {
  description = "Name of the resource group"
  default     = "tests-rg"
}

variable "cluster_name" {
  description = "Name of the AKS cluster"
  default     = "tests-aks"
}

variable "node_count" {
  description = "Number of nodes in the cluster"
  default     = 1
}

variable "test_scenario" {
  description = "Test scenario to differentiate state files"
  type        = string
}

variable "run_id" {
  description = "GitHub run ID for uniquely identifying state files"
  type        = string
}