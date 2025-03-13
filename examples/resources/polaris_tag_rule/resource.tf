# All RSC cloud accounts matching all my-key tags.
resource "polaris_tag_rule" "rule" {
  name           = "my-tag-rule"
  object_type    = "AWS_EC2_INSTANCE"
  tag_key        = "my-key"
  tag_all_values = true
}

# Specific RSC cloud account matching only the my-key tags with the value
# my-value.
data "polaris_aws_account" "account" {
  name = "my-aws-account"
}

resource "polaris_tag_rule" "rule" {
  name        = "my-tag-rule"
  object_type = "AWS_EC2_INSTANCE"
  tag_key     = "my-key"
  tag_value   = "my-value"

  cloud_account_ids = [
    data.polaris_aws_account.account.id,
  ]
}
