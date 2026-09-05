# Free-tier deployment: one EC2 instance running the same docker-compose
# stack that runs locally, plus an S3 bucket for files.
#
# This is the sibling of ../terraform, which describes the ECS/Fargate
# architecture. That one is the better design; this one is the one that
# costs nothing. Both are real - they just make opposite trades, and the
# Fargate config stays in the repo as the documented "how this scales"
# answer rather than pretending a single box is a production architecture.
#
# What actually stays inside the 12-month free tier:
#   EC2      750 hrs/month of t3.micro  -> one instance running 24/7
#   EBS      30 GB gp3                  -> 20 GB root volume here
#   S3       5 GB + 20k GET / 2k PUT
#   Data out 100 GB/month
#
# Postgres and Redis run as containers on the instance rather than RDS and
# ElastiCache, because those two are what the Fargate config spends real
# money on once their own free tiers lapse after 12 months.

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.region

  default_tags {
    tags = {
      Project     = "fileforge"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

locals {
  name = "fileforge-${var.environment}"
}

# Amazon Linux 2023, resolved from SSM so the AMI ID is never hardcoded -
# AMI IDs differ per region and change with every release.
data "aws_ssm_parameter" "al2023" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-6.1-x86_64"
}

# The default VPC is used deliberately: building a VPC here would add
# nothing except the chance of an unused NAT gateway quietly billing.
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}
