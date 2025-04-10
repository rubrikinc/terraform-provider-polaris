# Match all AWS EC2 instances which has a tag called my-key in all RSC cloud
# accounts.
resource "polaris_tag_rule" "rule" {
  name           = "my-tag-rule"
  object_type    = "AWS_EC2_INSTANCE"
  tag_key        = "my-key"
  tag_all_values = true
}

# Match all Azure VMs which has a tag called my-key with the value my-value in
# the my-azure-subscription RSC cloud account.
data "polaris_azure_subscription" "subscription" {
  name = "my-azure-subscription"
}

resource "polaris_tag_rule" "rule" {
  name        = "my-tag-rule"
  object_type = "AZURE_VIRTUAL_MACHINE"
  tag_key     = "my-key"
  tag_value   = "my-value"

  cloud_account_ids = [
    data.polaris_azure_subscription.subscription.id,
  ]
}
