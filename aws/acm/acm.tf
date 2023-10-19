resource "aws_acm_certificate" "domain" {
  provider    = aws.us-east-1
  domain_name = var.domain
  subject_alternative_names = [
    "*.${var.domain}",
  ]
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
    # TF bug on AWS 2.0: prevents certificates from being destroyed/recreated
    # https://github.com/hashicorp/terraform-provider-aws/issues/8531
    ignore_changes = [subject_alternative_names]
  }

  tags = {
    CostCentre = "${var.product_name}-${var.env}"
    Terraform  = true
  }
}