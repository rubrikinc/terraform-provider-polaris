# Using hardcoded values.
resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"

  regions = [
    "us-east-2",
    "us-west-2",
  ]

  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "EXOCOMPUTE"
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]
  }
}

# Using variables for the account values and the features. The dynamic
# feature block could also be expanded from the polaris_aws_cnp_artifacts
# data source.
variable "name" {
  type        = string
  description = "AWS account name."
}

variable "native_id" {
  type        = string
  description = "AWS account ID."
}

variable "regions" {
  type        = set(string)
  description = "AWS regions to protect."
}

variable "features" {
  type = map(object({
    permission_groups = set(string)
  }))
  description = "RSC features with permission groups."
}

resource "polaris_aws_cnp_account" "account" {
  name      = var.name
  native_id = var.native_id
  regions   = var.regions

  dynamic "feature" {
    for_each = var.features
    content {
      name              = feature.key
      permission_groups = feature.value["permission_groups"]
    }
  }
}
