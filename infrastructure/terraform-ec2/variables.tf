variable "region" {
  description = "AWS region. Stick to one region - free tier allowances are account-wide, not per region, so spreading resources around just makes them harder to find."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name, used as a suffix on every resource."
  type        = string
  default     = "demo"
}

variable "instance_type" {
  description = "t3.micro and t2.micro are the free-tier eligible types (750 hrs/month for 12 months). Anything larger bills from the first hour."
  type        = string
  default     = "t3.micro"
}

variable "root_volume_gb" {
  description = "Root EBS volume. Free tier covers 30 GB; 20 leaves headroom for the LibreOffice image, which is by far the largest."
  type        = number
  default     = 20
}

variable "ssh_ingress_cidr" {
  description = "Who may SSH in. Defaults to nobody - set it to YOUR.IP.ADDR.ESS/32 to enable SSH. Leaving this at 0.0.0.0/0 exposes port 22 to the entire internet, which is how free-tier instances get taken over for crypto mining."
  type        = string
  default     = "127.0.0.1/32"
}

variable "key_pair_name" {
  description = "Name of an existing EC2 key pair for SSH access. Leave empty to launch without SSH - the instance is still reachable through SSM Session Manager, which needs no open port and no key."
  type        = string
  default     = ""
}

variable "repo_url" {
  description = "Public git URL the instance clones on boot."
  type        = string
  default     = "https://github.com/Janith-Bhashitha/fileforge.git"
}

variable "repo_branch" {
  description = "Branch to deploy."
  type        = string
  default     = "main"
}

variable "file_retention_days" {
  description = "S3 lifecycle expiry for uploaded files. Also keeps the bucket inside the 5 GB free tier by not letting it grow forever."
  type        = number
  default     = 7
}

variable "alert_email" {
  description = "Address to send billing alerts to. Leave empty to create the alarm without a subscription - you can add one in the console later. AWS emails a confirmation link that must be clicked before alerts are delivered."
  type        = string
  default     = ""
}

variable "billing_alarm_threshold_usd" {
  description = "Fire the billing alarm above this many dollars. Free-tier usage should sit at or near zero, so a low threshold is a real signal rather than noise."
  type        = number
  default     = 5
}
