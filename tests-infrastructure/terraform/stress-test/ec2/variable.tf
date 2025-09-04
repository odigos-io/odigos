# =============================================================================
# BASIC CONFIGURATION
# =============================================================================

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "odigos-stress-test"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

# =============================================================================
# INSTANCE CONFIGURATION
# =============================================================================

variable "instance_type" {
  description = "EC2 instance type for monitoring stack"
  type        = string
  default     = "m6i.large"
  validation {
    condition     = can(regex("^[a-z][0-9][a-z]\\.[a-z]+$", var.instance_type))
    error_message = "Instance type must be a valid AWS instance type (e.g., m6i.large)."
  }
}

variable "root_volume_size" {
  description = "Size of the root volume in GB"
  type        = number
  default     = 40
}

variable "prometheus_volume_size" {
  description = "Size of the Prometheus data volume in GB"
  type        = number
  default     = 40
}

variable "grafana_volume_size" {
  description = "Size of the Grafana data volume in GB"
  type        = number
  default     = 1
}

variable "clickhouse_volume_size" {
  description = "Size of the ClickHouse data volume in GB"
  type        = number
  default     = 100
}

# =============================================================================
# SOFTWARE VERSIONS
# =============================================================================

variable "prometheus_version" {
  description = "Version of Prometheus to install"
  type        = string
  default     = "2.52.0"
}

variable "k6_version" {
  description = "Version of K6 to install"
  type        = string
  default     = "0.51.0"
  validation {
    condition     = can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+$", var.k6_version))
    error_message = "K6 version must be in format X.Y.Z."
  }
}

# =============================================================================
# APPLICATION CONFIGURATION
# =============================================================================

variable "k6_target_test_service_url" {
  type    = string
  default = ""
}
