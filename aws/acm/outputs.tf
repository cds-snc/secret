output acm_cert_name_validation {
  description = "Certificate verification CNAME for naked domain"
  value       = aws_acm_certificate.domain.domain_validation_options
}

output domain_cert_arn {
  value = aws_acm_certificate.domain.arn
}