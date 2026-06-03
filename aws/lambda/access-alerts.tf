data "aws_partition" "current" {}

locals {
  access_alerts_trail_name             = "${var.product_name}-${var.env}-access-alerts"
  access_alerts_cloudtrail_bucket_name = lower("${var.product_name}-${var.env}-${data.aws_caller_identity.current.account_id}-${var.region}-cloudtrail")
  api_lambda_assumed_role_arn_prefix   = "arn:${data.aws_partition.current.partition}:sts::${data.aws_caller_identity.current.account_id}:assumed-role/${var.product_name}-${var.env}-api/"
  cloudtrail_log_expiration_days       = 90
  cloudtrail_log_noncurrent_days       = 90
  eventbridge_management_events_state  = "ENABLED_WITH_ALL_CLOUDTRAIL_MANAGEMENT_EVENTS"

  # CloudTrail needs its S3 and KMS policies before the trail is created, so this
  # cannot reference aws_cloudtrail.access_alerts.arn without creating a cycle.
  access_alerts_expected_trail_arn = "arn:${data.aws_partition.current.partition}:cloudtrail:${var.region}:${data.aws_caller_identity.current.account_id}:trail/${local.access_alerts_trail_name}"

  access_alerts_tags = {
    Name       = local.access_alerts_trail_name
    CostCenter = "${var.product_name}-${var.env}"
  }
}

data "aws_sns_topic" "internal_sre_alert" {
  name = "internal-sre-alert"
}

data "aws_iam_policy_document" "access_alerts_cloudtrail_kms" {
  # checkov:skip=CKV_AWS_109:KMS key policies require the account root principal to enable IAM administration; CloudTrail service access is SourceArn scoped.
  # checkov:skip=CKV_AWS_111:KMS key policies require the account root principal to enable IAM administration; CloudTrail write access is SourceArn and encryption-context scoped.
  statement {
    sid    = "EnableIAMUserPermissions"
    effect = "Allow"

    principals {
      type        = "AWS"
      identifiers = ["arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"]
    }

    actions   = ["kms:*"]
    resources = ["*"]
  }

  statement {
    sid    = "AllowCloudTrailDescribeKey"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["cloudtrail.amazonaws.com"]
    }

    actions   = ["kms:DescribeKey"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "aws:SourceArn"
      values   = [local.access_alerts_expected_trail_arn]
    }
  }

  statement {
    sid    = "AllowCloudTrailEncryptLogs"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["cloudtrail.amazonaws.com"]
    }

    actions   = ["kms:GenerateDataKey*"]
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "aws:SourceArn"
      values   = [local.access_alerts_expected_trail_arn]
    }

    condition {
      test     = "StringLike"
      variable = "kms:EncryptionContext:aws:cloudtrail:arn"
      values   = [local.access_alerts_expected_trail_arn]
    }
  }
}

resource "aws_kms_key" "access_alerts_cloudtrail" {
  description             = "${var.product_name}-${var.env} access alerts CloudTrail log key"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  policy                  = data.aws_iam_policy_document.access_alerts_cloudtrail_kms.json

  tags = local.access_alerts_tags
}

resource "aws_kms_alias" "access_alerts_cloudtrail" {
  name          = "alias/${var.product_name}-${var.env}-access-alerts-cloudtrail"
  target_key_id = aws_kms_key.access_alerts_cloudtrail.key_id
}

module "access_alerts_cloudtrail_bucket" {
  source = "github.com/cds-snc/terraform-modules//S3?ref=v9.6.8"

  bucket_name       = local.access_alerts_cloudtrail_bucket_name
  billing_tag_value = var.product_name

  versioning = {
    enabled = true
  }

  lifecycle_rule = [
    {
      id      = "expire-cloudtrail-logs"
      enabled = true

      expiration = {
        days = local.cloudtrail_log_expiration_days
      }

      noncurrent_version_expiration = {
        days = local.cloudtrail_log_noncurrent_days
      }
    },
  ]

  tags = local.access_alerts_tags
}

resource "aws_cloudtrail" "access_alerts" {
  # checkov:skip=CKV_AWS_67:This trail is scoped to the app's single deployment region.
  # checkov:skip=CKV2_AWS_10:EventBridge consumes CloudTrail events directly for these alerts; CloudWatch Logs delivery would duplicate the S3 log archive.
  name                          = local.access_alerts_trail_name
  s3_bucket_name                = module.access_alerts_cloudtrail_bucket.s3_bucket_id
  include_global_service_events = false
  is_multi_region_trail         = false
  enable_log_file_validation    = true
  kms_key_id                    = aws_kms_key.access_alerts_cloudtrail.arn

  advanced_event_selector {
    name = "Unexpected DynamoDB table access"

    field_selector {
      field  = "eventCategory"
      equals = ["Data"]
    }

    field_selector {
      field  = "eventSource"
      equals = ["dynamodb.amazonaws.com"]
    }

    field_selector {
      field  = "resources.type"
      equals = ["AWS::DynamoDB::Table"]
    }

    field_selector {
      field  = "resources.ARN"
      equals = [aws_dynamodb_table.dynamodb-table.arn]
    }

    field_selector {
      field           = "userIdentity.arn"
      not_starts_with = [local.api_lambda_assumed_role_arn_prefix]
    }
  }

  advanced_event_selector {
    name = "KMS management events"

    field_selector {
      field  = "eventCategory"
      equals = ["Management"]
    }

    field_selector {
      field  = "eventSource"
      equals = ["kms.amazonaws.com"]
    }
  }

  tags = local.access_alerts_tags
}

resource "aws_cloudwatch_event_rule" "unexpected_dynamodb_access" {
  name        = "${var.product_name}-${var.env}-unexpected-dynamodb-access"
  description = "Detect unexpected access to the ${aws_dynamodb_table.dynamodb-table.name} DynamoDB table"

  event_pattern = jsonencode({
    source      = ["aws.dynamodb"]
    detail-type = ["AWS API Call via CloudTrail"]
    detail = {
      eventSource = ["dynamodb.amazonaws.com"]
      resources = {
        ARN = [aws_dynamodb_table.dynamodb-table.arn]
      }
      userIdentity = {
        arn = [
          {
            anything-but = {
              prefix = local.api_lambda_assumed_role_arn_prefix
            }
          },
        ]
      }
    }
  })
}

resource "aws_cloudwatch_event_target" "unexpected_dynamodb_access" {
  rule      = aws_cloudwatch_event_rule.unexpected_dynamodb_access.name
  target_id = "InternalSREAlert"
  arn       = data.aws_sns_topic.internal_sre_alert.arn

  input_transformer {
    input_paths = {
      event_name = "$.detail.eventName"
      event_time = "$.detail.eventTime"
      principal  = "$.detail.userIdentity.arn"
      region     = "$.detail.awsRegion"
      resource   = "$.detail.resources[0].ARN"
      source_ip  = "$.detail.sourceIPAddress"
    }

    input_template = "\"Unexpected DynamoDB access in ${var.product_name}-${var.env}\\nEvent: <event_name>\\nPrincipal: <principal>\\nSource IP: <source_ip>\\nRegion: <region>\\nResource: <resource>\\nTime: <event_time>\""
  }
}

resource "aws_cloudwatch_event_rule" "unexpected_kms_access" {
  name        = "${var.product_name}-${var.env}-unexpected-kms-access"
  description = "Detect unexpected access to the ${var.product_name}-${var.env} key"
  state       = local.eventbridge_management_events_state

  event_pattern = jsonencode({
    source      = ["aws.kms"]
    detail-type = ["AWS API Call via CloudTrail"]
    detail = {
      eventSource = ["kms.amazonaws.com"]
      resources = {
        ARN = [aws_kms_key.key.arn]
      }
      userIdentity = {
        arn = [
          {
            anything-but = {
              prefix = local.api_lambda_assumed_role_arn_prefix
            }
          },
        ]
      }
    }
  })
}

resource "aws_cloudwatch_event_target" "unexpected_kms_access" {
  rule      = aws_cloudwatch_event_rule.unexpected_kms_access.name
  target_id = "InternalSREAlert"
  arn       = data.aws_sns_topic.internal_sre_alert.arn

  input_transformer {
    input_paths = {
      event_name = "$.detail.eventName"
      event_time = "$.detail.eventTime"
      principal  = "$.detail.userIdentity.arn"
      region     = "$.detail.awsRegion"
      resource   = "$.detail.resources[0].ARN"
      source_ip  = "$.detail.sourceIPAddress"
    }

    input_template = "\"Unexpected KMS access in ${var.product_name}-${var.env}\\nEvent: <event_name>\\nPrincipal: <principal>\\nSource IP: <source_ip>\\nRegion: <region>\\nResource: <resource>\\nTime: <event_time>\""
  }
}
