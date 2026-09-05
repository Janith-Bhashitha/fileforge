variable "region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name, used as a suffix on every resource."
  type        = string
  default     = "staging"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "db_username" {
  description = "RDS master username."
  type        = string
  default     = "fileforge"
}

variable "db_password" {
  description = "RDS master password. Supply via TF_VAR_db_password or a tfvars file kept out of git - never a default."
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "Signing secret for API tokens. Supply via TF_VAR_jwt_secret."
  type        = string
  sensitive   = true
}

variable "db_instance_class" {
  description = "RDS instance size. Deliberately small: this is a portfolio deployment, not a production fleet."
  type        = string
  default     = "db.t4g.micro"
}

variable "file_retention_days" {
  description = "How long uploaded objects live before S3 lifecycle expires them. Matches the app's RETENTION_DAYS."
  type        = number
  default     = 7
}

variable "api_image" {
  description = "ECR image URI for the API. Set by CI to the commit SHA tag."
  type        = string
  default     = ""
}
