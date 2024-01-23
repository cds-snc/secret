data "archive_file" "lambda-warmer" {
  type        = "zip"
  source_file = "${path.module}/src/lambda-warmer.py"
  output_path = "/tmp/lambda-warmer.zip"
}

resource "aws_iam_role" "lambda-iam-role" {
  name               = "${var.product_name}-${var.env}-lambda-iam-role"
  assume_role_policy = <<EOF
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Sid":"",
         "Effect":"Allow",
         "Principal":{
            "Service":"lambda.amazonaws.com"
         },
         "Action":"sts:AssumeRole"
      }
   ]
}
EOF
}

resource "aws_lambda_function" "lambda-warmer" {
  filename         = data.archive_file.lambda-warmer.output_path
  function_name    = "${var.product_name}-${var.env}-lambda-warmer"
  handler          = "lambda-warmer.handler"
  memory_size      = 128
  role             = aws_iam_role.lambda-iam-role.arn
  runtime          = "python3.8"
  source_code_hash = data.archive_file.lambda-warmer.output_base64sha256
  timeout          = 60

  environment {
    variables = {
      BASE_URL = "https://${var.domain}"
    }
  }

  tags = {
    Name       = var.product_name
    CostCenter = "${var.product_name}-${var.env}"
  }
}


resource "aws_cloudwatch_event_rule" "every-three-minutes" {
  name                = "lambda-warmer"
  description         = "Fires every three minutes"
  schedule_expression = "rate(3 minutes)"
}

resource "aws_cloudwatch_event_target" "tigger-lambda-every-three-minutes" {
  rule      = aws_cloudwatch_event_rule.every-three-minutes.name
  target_id = "${var.product_name}-${var.env}-lambda-warmer"
  arn       = aws_lambda_function.lambda-warmer.arn
}

resource "aws_lambda_permission" "allow-cloudwatch-to-call-lambda" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda-warmer.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every-three-minutes.arn
}
