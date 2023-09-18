resource "polaris_aws_cnp_account_trust_policy" "trust_policy" {
  for_each    = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  account_id  = polaris_aws_cnp_account.account.id
  features    = polaris_aws_cnp_account.account.features
  external_id = polaris_aws_cnp_account.account.external_id
  role_key    = each.key
}
