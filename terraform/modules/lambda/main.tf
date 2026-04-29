# IAM Role for Lambda execution
resource "aws_iam_role" "lambda" {
  name = "${var.project_name}-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# IAM policy for DynamoDB access
resource "aws_iam_role_policy" "dynamodb" {
  name = "${var.project_name}-dynamodb-policy"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem"
      ]
      Resource = aws_dynamodb_table.data.arn
    }]
  })
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.project_name}"
  retention_in_days = 7

  tags = {
    Project = var.project_name
  }
}

# ECR Repository for Lambda container image
resource "aws_ecr_repository" "lambda" {
  name                 = "${var.project_name}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Project = var.project_name
  }
}

resource "aws_ecr_lifecycle_policy" "lambda" {
  repository = aws_ecr_repository.lambda.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 3 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 3
      }
      action = {
        type = "expire"
      }
    }]
  })
}

# DynamoDB table for application data (single-table design)
resource "aws_dynamodb_table" "data" {
  name         = "${var.project_name}-data"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"

  attribute {
    name = "pk"
    type = "S"
  }

  tags = {
    Project = var.project_name
  }
}

# Lambda Function
resource "aws_lambda_function" "api" {
  function_name = var.project_name
  role          = aws_iam_role.lambda.arn
  package_type  = "Image"
  image_uri     = var.bootstrap_image_uri
  timeout       = 30
  memory_size   = 512

  environment {
    variables = {
      PORT              = "8080"
      STORE_DRIVER      = "dynamodb"
      DYNAMODB_TABLE    = aws_dynamodb_table.data.name
      AWS_REGION_CUSTOM = var.aws_region
    }
  }

  # Policies must be attached before Lambda is created to avoid permission
  # errors on first invocation. Log group must exist first so AWS doesn't
  # auto-create it without the retention setting.
  depends_on = [
    aws_iam_role_policy_attachment.lambda_basic,
    aws_iam_role_policy.dynamodb,
    aws_cloudwatch_log_group.lambda,
  ]

  # image_uri is managed by CI (aws lambda update-function-code after each
  # push). Ignoring it here prevents Terraform from fighting CI and allows
  # the first apply to succeed before any image exists in ECR.
  lifecycle {
    ignore_changes = [image_uri]
  }

  tags = {
    Project = var.project_name
  }
}

# Lambda Function URL (public HTTPS endpoint)
resource "aws_lambda_function_url" "api" {
  function_name      = aws_lambda_function.api.function_name
  authorization_type = "NONE"

  cors {
    allow_origins     = ["*"]
    allow_methods     = ["*"]
    allow_headers     = ["*"]
    expose_headers    = ["*"]
    max_age           = 86400
  }
}
