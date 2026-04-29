variable "project_name" {
  description = "Project name for resource naming"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "bootstrap_image_uri" {
  description = "Image URI used only on first terraform apply, before CI pushes the real image. Subsequent deploys are handled by aws lambda update-function-code."
  type        = string
  default     = "public.ecr.aws/lambda/provided:al2023"
}
