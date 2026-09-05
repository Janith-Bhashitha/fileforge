terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
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

  # Every service runs the same image family and differs only in the command
  # and how far it scales, so they're defined as data rather than four
  # near-identical resource blocks.
  workers = {
    "worker-pdf"    = { cpu = 512, memory = 1024, count = 1 }
    "worker-image"  = { cpu = 512, memory = 1024, count = 1 }
    "worker-office" = { cpu = 1024, memory = 2048, count = 1 } # LibreOffice is the heavy one
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}
