# Output the IP addresses and version used by the RSC deployment.
data "polaris_deployment" "deployment" {}

output "ip_addresses" {
  value = data.polaris_deployment.deployment.ip_addresses
}

output "version" {
  value = data.polaris_deployment.deployment.version
}
