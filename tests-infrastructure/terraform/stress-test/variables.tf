variable "cluster_name" {
  description = "Name of the EKS cluster"
  default     = "stress-tests-eks"
}

variable "region" {
  description = "AWS region to deploy the cluster in"
  default     = "us-east-1"
}

variable "node_count" {
  description = "Number of nodes in the cluster"
  default     = 5
}

variable "node_spec" {
  description = "The node spec for the cluster"
  default = "c6a.xlarge" 
}

variable "monitoring_sg_id" {
  description = "Security Group ID of the EC2 monitoring instance"
  type        = string
}