variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "gymshark-challenge"
}

variable "domain_name" {
  description = "Custom domain name (optional, for documentation)"
  type        = string
  default     = ""
}

variable "bootstrap_image_uri" {
  description = "Image URI for first Lambda deploy, before CI pushes the real image"
  type        = string
  default     = ""
}

variable "certificate_arn" {
  description = "ACM certificate ARN for CloudFront custom domain (must be in us-east-1)"
  type        = string
  default     = ""
}
