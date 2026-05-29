terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # State is local by default (fine for one person). For shared/CI state, add an
  # S3 + DynamoDB backend here later.
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
