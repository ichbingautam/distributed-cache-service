# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "IDs of public subnets"
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "IDs of private subnets"
  value       = aws_subnet.private[*].id
}

output "ecr_repository_url" {
  description = "URL of the ECR repository for cache service"
  value       = aws_ecr_repository.cache_service.repository_url
}

output "ecs_cluster_name" {
  description = "Name of the ECS cluster"
  value       = aws_ecs_cluster.main.name
}

output "ecs_cluster_arn" {
  description = "ARN of the ECS cluster"
  value       = aws_ecs_cluster.main.arn
}

output "cache_service_url" {
  description = "URL of the cache service via ALB"
  value       = "http://${aws_lb.main.dns_name}"
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "Zone ID of the Application Load Balancer (for Route53)"
  value       = aws_lb.main.zone_id
}

output "service_discovery_namespace" {
  description = "Service discovery namespace for inter-service communication"
  value       = aws_service_discovery_private_dns_namespace.main.name
}

output "cache_service_discovery_endpoint" {
  description = "Service discovery endpoint for cache service"
  value       = "cache.${aws_service_discovery_private_dns_namespace.main.name}"
}

# Monitoring outputs (conditional)
output "grafana_url" {
  description = "URL to access Grafana dashboard"
  value       = var.enable_monitoring ? "http://${aws_lb.main.dns_name}/grafana" : null
}

output "prometheus_url" {
  description = "URL to access Prometheus"
  value       = var.enable_monitoring ? "http://${aws_lb.main.dns_name}/prometheus" : null
}

# CI/CD helper outputs
output "docker_login_command" {
  description = "AWS CLI command to login to ECR"
  value       = "aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"
}

output "docker_push_command" {
  description = "Command to push image to ECR"
  value       = "docker push ${aws_ecr_repository.cache_service.repository_url}:latest"
}
