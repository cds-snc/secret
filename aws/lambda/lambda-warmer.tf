data "archive_file" "lambda-warmer" {
  type        = "zip"
  source_file = "${path.module}/src/lambda-warmer.py"
  output_path = "/tmp/lambda-warmer.py.zip"
}

resource "aws_lambda_function" "lambda-warmer" {
  filename         = "/tmp/lambda-prewarm.py.zip"
  function_name    = "${var.product_name}-${var.env}-lambda-warmer"
  handler          = "main.handler"
  memory_size      = 128
  role             = aws_iam_role.lambda-iam-role.arn
  runtime          = "python3.8"
  source_code_hash = filebase64sha256("/tmp/lambda-warmer.py.zip")
  timeout          = 60

  environment {
    variables = {
      BASE_URL = "https://secret.cdssandbox.xyz"
    }
  }

  tags = {
    Name       = var.product_name
    CostCenter = "${var.product_name}-${var.env}"
  }
}


resource "aws_cloudwatch_event_rule" "every-five-minutes" {
  name                = "lambda-warmer"
  description         = "Fires every five minutes"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "tigger-lambda-every-five-minutes" {
  rule      = aws_cloudwatch_event_rule.every-five-minutes.name
  target_id = "${var.product_name}-${var.env}-lambda-warmer"
  arn       = aws_lambda_function.lambda-warmer.arn
}

resource "aws_lambda_permission" "allow-cloudwatch-to-call-lambda" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda-warmer.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every-five-minutes.arn
}