data "polaris_aws_account" "example" {
  name = "example"
}

output "example_aws_account" {
  value = data.polaris_aws_account.example
}
