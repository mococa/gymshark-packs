output "ecr_repository_url" {
  description = "ECR repository URL for pushing Docker images"
  value       = aws_ecr_repository.lambda.repository_url
}

output "function_url" {
  description = "Lambda Function URL - use this as CNAME target in DNS"
  value       = aws_lambda_function_url.api.function_url
}

output "function_name" {
  description = "Lambda function name"
  value       = aws_lambda_function.api.function_name
}
