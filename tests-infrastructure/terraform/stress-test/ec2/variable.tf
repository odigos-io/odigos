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

variable "k6_version" {
  type    = string
  default = "0.51.0"
}

variable "k6_frontend_url" {
  type    = string
  default = "http://ae5f01a90ea3b448c87310f92f68cbce-568516596.us-east-1.elb.amazonaws.com:8080"
}
