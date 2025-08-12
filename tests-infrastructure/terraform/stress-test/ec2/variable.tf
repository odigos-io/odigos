variable "key_pair_name" {
  description = "Name of your EC2 SSH key pair"
  type        = string
  default= ""
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}