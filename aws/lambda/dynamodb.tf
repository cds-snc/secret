resource "aws_dynamodb_table" "dynamodb-table" {
  name           = "${var.product_name}-${var.env}-table"
  read_capacity  = 1
  write_capacity = 1
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = {
    Name       = var.product_name
    CostCenter = "${var.product_name}-${var.env}"
  }
}