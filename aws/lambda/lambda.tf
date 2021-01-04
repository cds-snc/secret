data "template_file" "chalice_api_swagger" {
  template = file("${path.module}/swagger.json")

  vars = {
    "invoke_arn" = aws_lambda_function.api_handler.invoke_arn
  }
}

resource "aws_api_gateway_deployment" "rest-api" {
  lifecycle {
    create_before_destroy = true
  }

  rest_api_id       = aws_api_gateway_rest_api.rest-api.id
  stage_description = md5(data.template_file.chalice_api_swagger.rendered)
  stage_name        = "api"
}

resource "aws_api_gateway_rest_api" "rest-api" {
  name = "${var.product_name}-${var.env}-rest-api"
  binary_media_types = [
    "application/octet-stream",
    "application/x-tar",
    "application/zip",
    "audio/basic",
    "audio/ogg",
    "audio/mp4",
    "audio/mpeg",
    "audio/wav",
    "audio/webm",
    "image/png",
    "image/jpg",
    "image/jpeg",
    "image/gif",
    "video/ogg",
    "video/mpeg",
    "video/webm"
  ]

  body = data.template_file.chalice_api_swagger.rendered

  endpoint_configuration {
    types = ["EDGE"]
  }
}

resource "aws_api_gateway_domain_name" "rest-api" {
  certificate_arn = var.domain_cert_arn
  domain_name     = var.domain
}

resource "aws_api_gateway_base_path_mapping" "rest-api" {
  api_id      = aws_api_gateway_rest_api.rest-api.id
  stage_name  = aws_api_gateway_deployment.rest-api.stage_name
  domain_name = aws_api_gateway_domain_name.rest-api.domain_name
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

resource "aws_iam_role_policy" "lambda-iam-role-policy" {
  name = "${var.product_name}-${var.env}-lambda-iam-role-policy"
  role = aws_iam_role.lambda-iam-role.id

  policy = <<EOF
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Effect":"Allow",
         "Action":[
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents"
         ],
         "Resource":"arn:*:logs:*:*:*"
      },
      {
         "Effect":"Allow",
         "Action":[
            "kms:Decrypt",
            "kms:GenerateDataKey"
         ],
         "Resource":"${aws_kms_key.key.arn}"
      },
      {
         "Effect":"Allow",
         "Action":[
            "dynamodb:Query",
            "dynamodb:Scan",
            "dynamodb:GetItem",
            "dynamodb:PutItem",
            "dynamodb:UpdateItem",
            "dynamodb:DeleteItem"
         ],
         "Resource":"${aws_dynamodb_table.dynamodb-table.arn}"
      },
      {
         "Effect":"Allow",
         "Action":[
            "xray:PutTraceSegments",
            "xray:PutTelemetryRecords",
         ],
         "Resource":[
            "*"
         ]
      }
   ]
}
EOF
}

resource "aws_lambda_function" "api_handler" {
  filename         = "${path.module}/deployment.zip"
  function_name    = "${var.product_name}-${var.env}-lambda"
  handler          = "app.app"
  memory_size      = 128
  role             = aws_iam_role.lambda-iam-role.arn
  runtime          = "python3.8"
  source_code_hash = filebase64sha256("${path.module}/deployment.zip")
  timeout          = 60

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      DYNAMO_TABLE            = aws_dynamodb_table.dynamodb-table.name
      ENV                     = "PRODUCTION"
      KMS_ID                  = aws_kms_key.key.id
      POWERTOOLS_SERVICE_NAME = "secret"
    }
  }

  tags = {
    Name       = var.product_name
    CostCenter = "${var.product_name}-${var.env}"
  }
}

resource "aws_lambda_permission" "rest-api-invoke" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.rest-api.execution_arn}/*"
}