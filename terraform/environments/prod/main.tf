terraform {
  required_version = ">= 1.0"

  # Backend config loaded from backend.hcl or -backend-config flags
  backend "s3" {}

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

module "lambda" {
  source = "../../modules/lambda"

  project_name        = var.project_name
  aws_region          = var.aws_region
  bootstrap_image_uri = var.bootstrap_image_uri
}
