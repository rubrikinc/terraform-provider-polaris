---
page_title: "Upgrade Guide: v1.2.0"
---

# Upgrade Guide v1.2.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note, deprecated resources and fields will be removed in a future release, please migrate your configurations
to use the recommended replacements as soon as possible.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.2.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.2.0"
    }
  }
}
```
Next, upgrade the provider to the new version by running:
```bash
terraform init -upgrade
```
After the provider has been updated, validate the correctness of the Terraform configuration files by running:
```bash
terraform plan
```
If you get an error or an unwanted diff, please see the _Significant Changes and New Features_ below for additional
instructions. Otherwise, proceed by running:
```bash
terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.2.0 version.

## New Features

### Data Scanning Cyber Assisted Recovery

Support for Data Scanning Cyber Assisted Recovery has been added to the `polaris_aws_account` resource. The feature can
be enabled by adding the `cyber_recovery_data_scanning` block to the `polaris_aws_account` resource. Here's a simple
example showing how to enable the feature:
```terraform
resource "polaris_aws_account" "default" {
  profile = "default"

  cyber_recovery_data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  outpost {
    outpost_account_id      = "123456789123"
    outpost_account_profile = "outpost"

    permission_groups = [
      "BASIC",
    ]
  }
}
```

### AWS DynamoDB

Support for Cloud Native DynamoDB Protection has been added to the IAM roles workflow. The feature is enabled using the
`polaris_aws_cnp_account` and `polaris_aws_cnp_account_attachments` resources. Please refer to the
[aws_cnp_account](https://github.com/rubrikinc/terraform-provider-polaris-examples/tree/main/aws_cnp_account) example
for an example of how to onboard an AWS account with a set of RSC features. In addition, the `polaris_tag_rule` resource
has been updated to support assigning SLA domains to DynamoDB tables using the `AWS_DYNAMODB_TABLE` object type.

### Terraform Import

Support for importing Terraform resources, using either the `import` command or the `import` block, has been added for
the following resources:

 * [polaris_aws_account](../resources/aws_account.md)
 * [polaris_aws_archival_location](../resources/aws_archival_location.md)
 * [polaris_aws_cnp_account](../resources/aws_cnp_account.md)
 * [polaris_aws_cnp_account_attachments](../resources/aws_cnp_account_attachments.md)
 * [polaris_aws_cnp_account_trust_policy](../resources/aws_cnp_account_trust_policy.md)
 * [polaris_aws_custom_tags](../resources/aws_custom_tags.md)
 * [polaris_aws_exocompute](../resources/aws_exocompute.md)
 * [polaris_aws_private_container_registry](../resources/aws_private_container_registry.md)
 * [polaris_azure_archival_location](../resources/azure_archival_location.md)
 * [polaris_azure_custom_tags](../resources/azure_custom_tags.md)
 * [polaris_azure_exocompute](../resources/azure_exocompute.md)
 * [polaris_azure_private_container_registry](../resources/azure_private_container_registry.md)
 * [polaris_azure_subscription](../resources/azure_subscription.md)
 * [polaris_custom_role](../resources/custom_role.md)
 * [polaris_data_center_aws_account](../resources/data_center_aws_account.md)
 * [polaris_data_center_azure_subscription](../resources/data_center_azure_subscription.md)
 * [polaris_gcp_project](../resources/gcp_project.md)
 * [polaris_role_assignment](../resources/role_assignment.md)
 * [polaris_sla_domain_assignment](../resources/sla_domain_assignment.md)
 * [polaris_tag_rule](../resources/tag_rule.md)
 * [polaris_user](../resources/user.md)

Note, some of the resources require a special kind of resource ID to be imported, please refer to the documentation of
each resource for more information.

## Significant Changes

### Trust Policies

The `id` field of the `polaris_cnp_account_trust_policies` resource has changed. The field now contains a combination
of the role key and the RSC cloud account ID. Previously, the field contained just the RSC cloud account ID. This is a
breaking change if a configuration expects the `id` field to contain just the RSC cloud account ID. To work around this
issue, use the `id` field of the `polaris_aws_cnp_account` resource instead.

The `features` field of the `polaris_aws_cnp_account_trust_policy` resource has been deprecated. The field has no
replacement and is no longer used by the provider. If the `features` field is used in a configuration, Terraform will
output a warning similar to this:
```console
╷
│ Warning: Argument is deprecated
│ 
│   with polaris_aws_cnp_account_trust_policy.trust_policy["CROSSACCOUNT"],
│   on main.tf line 65, in resource "polaris_aws_cnp_account_trust_policy" "trust_policy":
│   65:   features    = keys(var.features)
│ 
│ no longer used by the provider, any value set is ignored.
╵
```
Removing the `features` field from the `polaris_cnp_account_trust_policy` should be safe and only result in an in-place
update of the resource:
```console
# polaris_aws_cnp_account_trust_policy.trust_policy["CROSSACCOUNT"] will be updated in-place
~ resource "polaris_aws_cnp_account_trust_policy" "trust_policy" {
    ~ features    = [
        - "CLOUD_NATIVE_ARCHIVAL",
        - "CLOUD_NATIVE_PROTECTION",
        - "EXOCOMPUTE",
      ]
      id          = "CROSSACCOUNT-7d1d123f-2561-4e76-9e73-523cb6a36dd7"
      # (4 unchanged attributes hidden)
  }
```
The `features` field of the `polaris_aws_cnp_account` resource will be removed in a future release.

A new `trust_policies` computed field has been added to the `polaris_aws_cnp_account` resource. The `trust_policies`
field contains the IAM trust policies for all the role keys required by the RSC features enabled for the AWS account.
The trust policies of the `trust_policies` field can be used instead of the `polaris_aws_cnp_account_trust_policy`
resource, reducing the overall number of resources needed to onboard an AWS account. When refreshing the state of the
configuration, an in-place update of the resource will occur:
```console
# polaris_aws_cnp_account.account has changed
~ resource "polaris_aws_cnp_account" "account" {
    id                          = "2977b8c6-9687-4622-920b-2388ab896e6f"
    name                        = "my-aws-account"
    + trust_policies              = [
        + {
            + policy   = jsonencode(
                {
                    + Statement = [
                        + {
                            + Action    = [
                                + "sts:AssumeRole",
                            ]
                            + Effect    = "Allow"
                            + Principal = {
                                + Service = "ec2.amazonaws.com"
                            }
                            + Sid       = "WorkerNodeAssumeRolePolicyDocumentSid"
                        },
                    ]
                    + Version   = "2012-10-17"
                }
            )
            + role_key = "EXOCOMPUTE_EKS_WORKERNODE"
        },
        + {
            + policy   = jsonencode(
                {
                    + Statement = [
                        + {
                            + Action    = [
                                + "sts:AssumeRole",
                            ]
                            + Effect    = "Allow"
                            + Principal = {
                                + Service = "eks.amazonaws.com"
                            }
                            + Sid       = "ClusterAssumeRolePolicyDocumentSid"
                        },
                    ]
                    + Version   = "2012-10-17"
                }
            )
            + role_key = "EXOCOMPUTE_EKS_MASTERNODE"
        },
        + {
            + policy   = jsonencode(
                {
                    + Statement = [
                        + {
                            + Action    = [
                                + "sts:AssumeRole",
                            ]
                            + Condition = {
                                + StringEquals = {
                                    + "sts:ExternalId" = [
                                        + "36dfdb2e-8095-4301-bef5-018d110ecbed",
                                    ]
                                }
                            }
                            + Effect    = "Allow"
                            + Principal = {
                                + AWS = "arn:aws:iam::522860868451:user/rsc-account-07ceb"
                            }
                        },
                    ]
                    + Version   = "2012-10-17"
                }
            )
            + role_key = "CROSSACCOUNT"
        },
    ]
    # (5 unchanged attributes hidden)

    # (3 unchanged blocks hidden)
}
```
The number of trust policies and their content depends on the RSC features enabled for the AWS account. The
`trust_policies` field will be re-computed every time the `features` field of the `polaris_aws_cnp_account` resource
changes.
