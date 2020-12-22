# DO NOT CHANGE ANYTHING BELOW HERE UNLESS YOU KNOW WHAT YOU ARE DOING

terraform {
  source = "../../aws//lambda"
}

dependencies {
  paths = ["../acm"]
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

include {
  path = find_in_parent_folders()
}

inputs = {
  domain_cert_arn     = dependency.acm.outputs.domain_cert_arn
}