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

variable "lambda_reserved_concurrent_executions" {
  description = "Maximum concurrent API Lambda executions to protect KMS and DynamoDB from abusive bursts."
  type        = number
  default     = 10

  validation {
    condition     = var.lambda_reserved_concurrent_executions >= 1
    error_message = "lambda_reserved_concurrent_executions must be at least 1."
  }
}

variable "encrypt_rate_limit" {
  description = "Maximum POST /encrypt requests allowed from one IP in a five-minute window."
  type        = number
  default     = 100

  validation {
    condition     = var.encrypt_rate_limit >= 10
    error_message = "encrypt_rate_limit must be at least 10."
  }
}

variable "decrypt_rate_limit" {
  description = "Maximum POST /decrypt requests allowed from one IP in a five-minute window."
  type        = number
  default     = 300

  validation {
    condition     = var.decrypt_rate_limit >= 10
    error_message = "decrypt_rate_limit must be at least 10."
  }
}
