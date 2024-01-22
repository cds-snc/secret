resource "aws_cloudfront_distribution" "api" {
  enabled     = true
  aliases     = [var.domain]
  price_class = "PriceClass_100"

  origin {
    domain_name = split("/", aws_lambda_function_url.api.function_url)[2]
    origin_id   = module.api.function_name

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_read_timeout    = 60
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }

    custom_header {
      name  = "X-CloudFront-Header"
      value = module.api.function_name
    }
  }

  # Optimized caching for all GET/HEAD requests
  default_cache_behavior {
    allowed_methods = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods  = ["GET", "HEAD"]

    forwarded_values {
      query_string = true
      cookies {
        forward = "none"
      }
    }

    target_origin_id           = module.api.function_name
    viewer_protocol_policy     = "redirect-to-https"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security_headers_api.id

    min_ttl     = 1
    default_ttl = 86400    # 24 hours
    max_ttl     = 31536000 # 365 days
    compress    = true
  }

  # Prevent caching of version calls
  ordered_cache_behavior {
    path_pattern    = "/version"
    allowed_methods = ["GET", "HEAD"]
    cached_methods  = ["GET", "HEAD"]

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    target_origin_id           = module.api.function_name
    viewer_protocol_policy     = "redirect-to-https"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security_headers_api.id

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
    compress    = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = var.domain_cert_arn
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = "sni-only"
  }

  tags = {
    CostCentre = var.product_name
    Terraform  = true
  }
}

resource "aws_cloudfront_response_headers_policy" "security_headers_api" {
  name = "api-cloudfront-headers"

  security_headers_config {
    frame_options {
      frame_option = "DENY"
      override     = true
    }
    content_type_options {
      override = true
    }
    referrer_policy {
      override        = true
      referrer_policy = "same-origin"
    }
    strict_transport_security {
      override                   = true
      access_control_max_age_sec = 31536000
      include_subdomains         = true
      preload                    = true
    }
    xss_protection {
      override   = true
      mode_block = true
      protection = true
    }
  }
}
