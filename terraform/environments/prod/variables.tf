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
