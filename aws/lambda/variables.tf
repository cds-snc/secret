variable "domain_cert_arn" {
  type = string
}

variable "ecr_arn" {
  type = string
}

variable "ecr_repository_url" {
  type = string
}

variable "git_sha" {
  type = string
}

variable "require_additional_password" {
  type    = bool
  default = false
}
