locals {
  account_id   = "637287734259"
  domain       = "encrypted-message.cdssandbox.xyz"
  env          = "production"
  product_name = "secret"
}

# DO NOT CHANGE ANYTHING BELOW HERE UNLESS YOU KNOW WHAT YOU ARE DOING

inputs = {
  account_id   = local.account_id
  domain       = local.domain
  env          = local.env
  product_name = local.product_name
  region       = "ca-central-1"
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite"
  contents  = <<EOF
provider "aws" {
  region              = var.region
  version             = "~> 2.0"
  allowed_account_ids = [var.account_id]
}

provider "aws" {
  alias               = "us-east-1"
  region              = "us-east-1"
  version             = "~> 2.0"
  allowed_account_ids = [var.account_id]
}
EOF
}

generate "common_variables" {
  path      = "common_variables.tf"
  if_exists = "overwrite"
  contents  = <<EOF
variable account_id {
  description = "(Required) The account ID to perform actions on."
  type        = string
}

variable domain {
  description = "(Required) Domain name to deploy to"
  type        = string
}

variable env {
  description = "The current running environment"
  type        = string
}

variable product_name {
  description = "(Required) The name of the product you are deploying."
  type        = string
}

variable region {
  description = "The current AWS region"
  type        = string
}
EOF
}

remote_state {
  backend = "s3"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    encrypt             = true
    bucket              = "${local.product_name}-${local.env}-tf"
    dynamodb_table      = "terraform-state-lock-dynamo"
    region              = "ca-central-1"
    key                 = "${path_relative_to_include()}/terraform.tfstate"
    s3_bucket_tags      = { CostCenter : "${local.product_name}-${local.env}" }
    dynamodb_table_tags = { CostCenter : "${local.product_name}-${local.env}" }
  }
}