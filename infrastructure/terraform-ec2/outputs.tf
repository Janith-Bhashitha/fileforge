output "app_url" {
  description = "Where the app will be reachable. Allow ~10 minutes after apply: the instance builds all four images on first boot, which is slow on a t3.micro."
  value       = "http://${aws_eip.app.public_ip}"
}

output "public_ip" {
  description = "Static public address of the instance."
  value       = aws_eip.app.public_ip
}

output "instance_id" {
  description = "Use with SSM Session Manager for shell access without opening SSH: aws ssm start-session --target <this>"
  value       = aws_instance.app.id
}

output "s3_bucket" {
  description = "Bucket holding uploaded and converted files."
  value       = aws_s3_bucket.files.id
}

output "boot_log_hint" {
  description = "If the site doesn't come up, read the provisioning log here."
  value       = "aws ssm start-session --target ${aws_instance.app.id} then: sudo tail -100 /var/log/cloud-init-output.log"
}

# Marked sensitive so they never print in normal output. Retrieve
# deliberately with: terraform output -raw jwt_secret
output "jwt_secret" {
  description = "Generated token-signing secret."
  value       = random_password.jwt_secret.result
  sensitive   = true
}

output "postgres_password" {
  description = "Generated password for the containerised Postgres."
  value       = random_password.postgres.result
  sensitive   = true
}
