locals {
  lambda_hostname = trimsuffix(
    trimprefix(module.lambda.function_url, "https://"),
    "/"
  )

  subdomain = var.domain_name != "" ? split(".", var.domain_name)[0] : ""
}

output "ecr_repository_url" {
  description = "ECR repository URL for pushing Docker images"
  value       = module.lambda.ecr_repository_url
}

output "function_url" {
  description = "Lambda Function URL - add CNAME record pointing to this domain"
  value       = module.lambda.function_url
}

output "function_name" {
  description = "Lambda function name for updates"
  value       = module.lambda.function_name
}

output "dns_setup_instructions" {
  description = "Instructions for custom domain DNS setup"
  value = var.domain_name != "" ? <<EOT
Add CNAME record in your DNS provider:

Type:   CNAME
Name:   ${local.subdomain}
Target: ${local.lambda_hostname}
TTL:    Auto or 300

Note: If using a DNS proxy (e.g., Cloudflare), use DNS-only mode for compatibility.
EOT
 : "No custom domain configured - using Lambda Function URL directly"
}
