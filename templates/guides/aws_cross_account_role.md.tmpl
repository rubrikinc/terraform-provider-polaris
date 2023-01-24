---
page_title: "AWS Cross Account Role"
---

# Managing AWS accounts using cross account roles
In v0.4.0 support for cross account roles have been added through the `assume_role` parameter. Because of this the
`profile` parameter is no longer marked as required, instead at least one of `profile` or `assume_role` must be
specified. In the case where only `assume_role` is specified, the default profile will be used to assume that role.
Most of the AWS CLI environment variables can be used to override aspects of the default profile.

## Profile
When only `profile` is specified, we add the AWS account identified by the profile to RSC. The profile will be stored
in the Terraform state. This is how it works in v0.3.0.

### Example Usage
```terraform
resource "polaris_aws_account" "account" {
  profile = "my-profile-for-account"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```

## Profile with role
When both `profile` and `assume_role` are specified, we use the profile to assume the role and add the AWS account
identified by the role (the trusting account) to RSC. Both the profile and the role will be stored in the Terraform
state.

### Example Usage
```terraform
resource "polaris_aws_account" "account" {
  profile     = "my-profile"
  assume_role = "arn:aws:iam::123456789012:role/MyCrossAccountRole"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```

## Role
When only `assume_role` is specified, we use the default profile to assume the role and add the AWS account identified
by the role (the trusting account) to RSC. Only the assumed role is stored in the Terraform state. There will be no
connection to the profile in the Terraform state, so when updating the configuration any profile can be used as long as
it has permission to assume the role in the Terraform state.

### Example Usage
```terraform
resource "polaris_aws_account" "account" {
  assume_role = "arn:aws:iam::123456789012:role/MyCrossAccountRole"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```
