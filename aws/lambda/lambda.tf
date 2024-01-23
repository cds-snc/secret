data "aws_iam_policy_document" "api_policies" {
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["arn:*:logs:*:*:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
    ]
    resources = [aws_kms_key.key.arn]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
    ]
    resources = [aws_dynamodb_table.dynamodb-table.arn]
  }
}

module "api" {
  source    = "github.com/cds-snc/terraform-modules//lambda?ref=v9.0.3"
  name      = "${var.product_name}-${var.env}-api"
  ecr_arn   = var.ecr_arn
  image_uri = "${var.ecr_repository_url}:latest"

  memory                 = 128
  timeout                = 60
  enable_lambda_insights = true

  policies = [
    aws_iam_policy_document.api_policies.json,
  ]

  environment_variables = {
    DYNAMO_TABLE = aws_dynamodb_table.dynamodb-table.name
    ENV          = "PRODUCTION"
    GIT_SHA      = var.git_sha
    KMS_ID       = aws_kms_key.key.id
  }

  billing_tag_value = var.product_name
}

resource "aws_lambda_function_url" "api" {
  function_name      = module.api.function_name
  authorization_type = "NONE"
}
