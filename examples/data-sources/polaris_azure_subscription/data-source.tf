data "polaris_azure_subscription" "example" {
  name = "example"
}

output "example_azure_subscription" {
  value = data.polaris_azure_subscription.example
}
