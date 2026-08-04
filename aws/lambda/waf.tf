resource "aws_wafv2_web_acl" "api" {
  # checkov:skip=CKV_AWS_192:Secret bodies are opaque data; payload-signature blocking would reject legitimate messages without protecting an interpreter.
  # checkov:skip=CKV2_AWS_47:Secret bodies are opaque data; payload-signature blocking would reject legitimate messages without protecting an interpreter.
  # checkov:skip=CKV2_AWS_31:Full WAF request logging is intentionally disabled to minimize secret metadata exposure; per-rule CloudWatch metrics remain enabled.
  provider = aws.us-east-1

  name  = "${var.product_name}-${var.env}-api"
  scope = "CLOUDFRONT"

  default_action {
    allow {}
  }

  association_config {
    request_body {
      cloudfront {
        default_size_inspection_limit = "KB_64"
      }
    }
  }

  rule {
    name     = "request-body-size-limit"
    priority = 0

    action {
      block {
        custom_response {
          response_code = 413
        }
      }
    }

    statement {
      size_constraint_statement {
        comparison_operator = "GT"
        size                = 65536

        field_to_match {
          body {
            oversize_handling = "MATCH"
          }
        }

        text_transformation {
          priority = 0
          type     = "NONE"
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.product_name}-${var.env}-request-body-size-limit"
      sampled_requests_enabled   = false
    }
  }

  rule {
    name     = "encrypt-rate-limit"
    priority = 1

    action {
      block {
        custom_response {
          response_code = 429
        }
      }
    }

    statement {
      rate_based_statement {
        aggregate_key_type = "IP"
        limit              = var.encrypt_rate_limit

        scope_down_statement {
          and_statement {
            statement {
              byte_match_statement {
                positional_constraint = "EXACTLY"
                search_string         = "/encrypt"

                field_to_match {
                  uri_path {}
                }

                text_transformation {
                  priority = 0
                  type     = "NONE"
                }
              }
            }

            statement {
              byte_match_statement {
                positional_constraint = "EXACTLY"
                search_string         = "POST"

                field_to_match {
                  method {}
                }

                text_transformation {
                  priority = 0
                  type     = "NONE"
                }
              }
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.product_name}-${var.env}-encrypt-rate-limit"
      sampled_requests_enabled   = false
    }
  }

  rule {
    name     = "decrypt-rate-limit"
    priority = 2

    action {
      block {
        custom_response {
          response_code = 429
        }
      }
    }

    statement {
      rate_based_statement {
        aggregate_key_type = "IP"
        limit              = var.decrypt_rate_limit

        scope_down_statement {
          and_statement {
            statement {
              byte_match_statement {
                positional_constraint = "STARTS_WITH"
                search_string         = "/decrypt/"

                field_to_match {
                  uri_path {}
                }

                text_transformation {
                  priority = 0
                  type     = "LOWERCASE"
                }
              }
            }

            statement {
              byte_match_statement {
                positional_constraint = "EXACTLY"
                search_string         = "POST"

                field_to_match {
                  method {}
                }

                text_transformation {
                  priority = 0
                  type     = "UPPERCASE"
                }
              }
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${var.product_name}-${var.env}-decrypt-rate-limit"
      sampled_requests_enabled   = false
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.product_name}-${var.env}-api"
    sampled_requests_enabled   = false
  }

  tags = {
    Name       = "${var.product_name}-${var.env}-api"
    CostCenter = "${var.product_name}-${var.env}"
  }
}
