# -----------------------------------------------------------------------------
# General Configuration
# -----------------------------------------------------------------------------

variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "app_name" {
  description = "Application name prefix for resources"
  type        = string
  default     = "distributed-cache"
}

# -----------------------------------------------------------------------------
# Cache Service Configuration
# -----------------------------------------------------------------------------

variable "cache_replicas" {
  description = "Number of cache service replicas"
  type        = number
  default     = 3

  validation {
    condition     = var.cache_replicas >= 1 && var.cache_replicas <= 10
    error_message = "Cache replicas must be between 1 and 10."
  }
}

variable "cache_cpu" {
  description = "CPU units for cache service (256 = 0.25 vCPU)"
  type        = number
  default     = 512
}

variable "cache_memory" {
  description = "Memory in MB for cache service"
  type        = number
  default     = 1024
}

variable "cache_image_tag" {
  description = "Docker image tag for cache service"
  type        = string
  default     = "latest"
}

# -----------------------------------------------------------------------------
# Networking Configuration
# -----------------------------------------------------------------------------

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "List of availability zones (leave empty to auto-detect)"
  type        = list(string)
  default     = []
}

# -----------------------------------------------------------------------------
# Monitoring Configuration
# -----------------------------------------------------------------------------

variable "enable_monitoring" {
  description = "Enable Prometheus and Grafana monitoring stack"
  type        = bool
  default     = true
}

variable "grafana_admin_password" {
  description = "Admin password for Grafana"
  type        = string
  default     = "admin"
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Tags
# -----------------------------------------------------------------------------

variable "tags" {
  description = "Additional tags for all resources"
  type        = map(string)
  default     = {}
}
