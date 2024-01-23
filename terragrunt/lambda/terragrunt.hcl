# DO NOT CHANGE ANYTHING BELOW HERE UNLESS YOU KNOW WHAT YOU ARE DOING

terraform {
  source = "../../aws//lambda"
}

dependencies {
  paths = ["../acm", "../ecr"]
}

dependency "acm" {
  config_path = "../acm"

  # Configure mock outputs for the `validate` command that are returned when there are no outputs available (e.g the
  # module hasn't been applied yet.
  mock_outputs_allowed_terraform_commands = ["plan-all", "validate"]
  mock_outputs = {
    domain_cert_arn     = ""
  }
}

dependency "ecr" {
  config_path = "../ecr"

  # Configure mock outputs for the `validate` command that are returned when there are no outputs available (e.g the
  # module hasn't been applied yet.
  mock_outputs_allowed_terraform_commands = ["plan-all", "validate"]
  mock_outputs = {
    ecr_arn = ""
    ecr_repository_url = ""
  }
}

include {
  path = find_in_parent_folders()
}

inputs = {
  domain_cert_arn     = dependency.acm.outputs.domain_cert_arn
  ecr_arn = dependency.ecr.outputs.ecr_arn
  ecr_repository_url = dependency.ecr.outputs.ecr_repository_url
}