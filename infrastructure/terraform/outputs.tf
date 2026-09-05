output "api_url" {
  description = "Public address of the API."
  value       = "http://${aws_lb.main.dns_name}"
}

output "s3_bucket" {
  description = "Bucket holding uploaded and converted files."
  value       = aws_s3_bucket.files.id
}

output "ecr_repositories" {
  description = "ECR repository URLs, keyed by service - CI pushes images here."
  value       = { for k, v in aws_ecr_repository.services : k => v.repository_url }
}

output "ecs_cluster" {
  description = "ECS cluster name, used by the deploy workflow."
  value       = aws_ecs_cluster.main.name
}

output "database_endpoint" {
  description = "RDS endpoint. Reachable only from inside the VPC."
  value       = aws_db_instance.main.endpoint
  sensitive   = true
}
