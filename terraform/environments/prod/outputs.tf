locals {
  lambda_hostname = trimsuffix(
    trimprefix(module.lambda.function_url, "https://"),
    "/"
  )

  subdomain = var.domain_name != "" ? split(".", var.domain_name)[0] : ""

  dns_instructions = var.domain_name != "" ? join("\n", [
    "Add CNAME record in your DNS provider:",
    "",
    "Type:   CNAME",
    "Name:   ${local.subdomain}",
    "Target: ${module.lambda.cloudfront_domain}",
    "TTL:    Auto or 300",
  ]) : "No custom domain configured - using Lambda Function URL directly"
}

output "ecr_repository_url" {
  description = "ECR repository URL for pushing Docker images"
  value       = module.lambda.ecr_repository_url
}

output "function_url" {
  description = "Lambda Function URL (direct access, no custom domain)"
  value       = module.lambda.function_url
}

output "function_name" {
  description = "Lambda function name for updates"
  value       = module.lambda.function_name
}

output "dns_setup_instructions" {
  description = "Instructions for custom domain DNS setup"
  value       = local.dns_instructions
}

output "cloudfront_domain" {
  description = "CloudFront domain to use as CNAME target (empty if CloudFront not enabled)"
  value       = module.lambda.cloudfront_domain
}
