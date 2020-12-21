output "EndpointURL" {
  value = aws_api_gateway_deployment.rest-api.invoke_url
}

output "RestAPIId" {
  value = aws_api_gateway_rest_api.rest-api.id
}

output "CustomDomainName" {
  value = aws_api_gateway_domain_name.rest-api.cloudfront_domain_name
}

output "CustomDomainZone" {
  value = aws_api_gateway_domain_name.rest-api.cloudfront_zone_id
}