# terraform

Infrastructure as Code for the AWS deployment.

- `bootstrap/` - one-time setup: S3 bucket and DynamoDB table for Terraform remote state.
- `environments/prod/` - production environment (Lambda, ECR, DynamoDB, optional CloudFront).
- `modules/lambda/` - reusable module: Lambda function, IAM roles, ECR repository, CloudWatch logs.

See the root [README](../README.md) for deployment instructions.
