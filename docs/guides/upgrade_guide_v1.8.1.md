---
page_title: "Upgrade Guide: v1.8.1"
---

# Upgrade Guide v1.8.1

The v1.8.1 release changes how the AWS IAM roles workflow surfaces the artifact for a role-chaining account: the
role-chaining role is now exposed under the `ROLE_CHAINING` artifact key instead of `CROSSACCOUNT`. The release also
adds support for the GCP `SERVERS_AND_APPS` feature. See the [changelog](changelog.md) for the full list.

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

Starting with v1.7.0, each release is also published as the renamed `rubrikinc/rubrik` provider. The
`rubrikinc/polaris` provider will continue to be released and supported for some time, so there is no need to switch
right now. The `rubrikinc/polaris` provider will eventually be retired, however, and you will need to switch to the
`rubrikinc/rubrik` provider before then. The migration paths will improve over time as more resources gain support for
Terraform's `moved {}` block, making the switch progressively simpler. See the
[latest upgrade guide for the rubrikinc/rubrik provider](https://registry.terraform.io/providers/rubrikinc/rubrik/latest/docs/guides)
for the currently available migration paths.

~> **Note:** If you are upgrading across multiple minor versions (e.g. v1.6.x to v1.8.1), review the upgrade guide for
each intermediate version as well. Each guide documents breaking changes and migration steps specific to that release.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.8.1 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.8.1"
    }
  }
}
```
Next, upgrade the provider to the new version by running:
```shell
% terraform init -upgrade
```
After the provider has been updated, validate the correctness of the Terraform configuration files by running:
```shell
% terraform plan
```
If you get an error or an unwanted diff, please see the _Significant Changes_ section below for additional instructions.
Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.8.1 version.

## Significant Changes

### AWS Role Chaining Artifact Key

Earlier releases surfaced the role-chaining role under the `CROSSACCOUNT` artifact key. It is now surfaced under the
`ROLE_CHAINING` key, which matches the account's feature. This affects only role-chaining accounts, that is accounts
whose sole feature is `ROLE_CHAINING`; other accounts are unchanged.

After upgrading, a role-chaining account shows a one-time diff that settles after a single `apply`. The
`polaris_aws_cnp_artifacts` data source returns `ROLE_CHAINING` in `role_keys` where it previously returned
`CROSSACCOUNT`, and the `polaris_aws_cnp_permissions` data source reports the policy under the `ROLE_CHAINING` artifact
with the policy name `RoleChainingPolicy` instead of `RoleChaining`. The permission policy document itself does not
change — it is still an `sts:AssumeRole` policy with the same statements, and the role's trust relationship is also
unchanged. Only the artifact key and the reported policy name differ.

The diff matters because the [AWS IAM roles workflow](aws_cnp_account.md) guide keys its AWS resources on these values:
`aws_iam_role` on `role_keys` and `aws_iam_policy` on the policy name. When those `for_each` keys change, Terraform
destroys and recreates the role-chaining `aws_iam_role` and its `aws_iam_policy` under new ARNs, re-attaches them, and
`polaris_aws_cnp_account_attachments` re-registers the role under the `ROLE_CHAINING` key with the recreated role's ARN.
The new role carries the same trust relationship and permissions as the old one.

No configuration change is required if your configuration iterates the artifacts and permissions data sources, as in
that guide — the data sources now emit `ROLE_CHAINING` automatically and the role keys follow. Review the plan, then
run `terraform apply` to let the diff settle.

A configuration change is only required where the `CROSSACCOUNT` key is hardcoded, for example as the `role_key` of a
`polaris_aws_cnp_permissions` data source. Update those references to `ROLE_CHAINING`:

```terraform
data "polaris_aws_cnp_permissions" "role_chaining" {
  role_key = "ROLE_CHAINING"

  feature {
    name              = "ROLE_CHAINING"
    permission_groups = ["BASIC"]
  }
}
```
