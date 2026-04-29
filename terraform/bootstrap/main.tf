terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# S3 bucket for Terraform state
resource "aws_s3_bucket" "tfstate" {
  bucket        = var.state_bucket_name
  force_destroy = true

  tags = {
    Name        = "Terraform State Bucket"
    Project     = "RE Partners Challenge"
    ManagedBy   = "Terraform"
  }
}

resource "aws_s3_bucket_versioning" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# DynamoDB table for state locking
resource "aws_dynamodb_table" "tfstate_lock" {
  name         = var.lock_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Name        = "Terraform State Lock Table"
    Project     = "RE Partners Challenge"
    ManagedBy   = "Terraform"
  }
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "domain_name" {
  description = "Custom domain for the app (e.g. gymshark-challenge.moureau.dev). If set, an ACM certificate is requested."
  type        = string
  default     = ""
}

variable "state_bucket_name" {
  description = "Name of the S3 bucket for Terraform state"
  type        = string
  default     = "moureau-tfstate-gymshark"
}

variable "lock_table_name" {
  description = "Name of the DynamoDB table for state locking"
  type        = string
  default     = "moureau-tfstate-lock"
}

output "state_bucket" {
  value = aws_s3_bucket.tfstate.id
}

output "lock_table" {
  value = aws_dynamodb_table.tfstate_lock.id
}

# ACM certificate for custom domain (only created when domain_name is set)
resource "aws_acm_certificate" "app" {
  count             = var.domain_name != "" ? 1 : 0
  domain_name       = var.domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Project   = "RE Partners Challenge"
    ManagedBy = "Terraform"
  }
}

output "certificate_arn" {
  description = "ACM certificate ARN — add as GitHub secret ACM_CERTIFICATE_ARN once the certificate is ISSUED"
  value       = var.domain_name != "" ? aws_acm_certificate.app[0].arn : "no domain configured"
}

output "certificate_validation_cname" {
  description = "Add this CNAME record to your DNS provider to validate the certificate"
  value = var.domain_name != "" ? {
    name  = tolist(aws_acm_certificate.app[0].domain_validation_options)[0].resource_record_name
    type  = tolist(aws_acm_certificate.app[0].domain_validation_options)[0].resource_record_type
    value = tolist(aws_acm_certificate.app[0].domain_validation_options)[0].resource_record_value
  } : null
}
