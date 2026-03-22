# AWS example: wait for an AWS account to be refreshed after onboarding.
resource "polaris_aws_account" "account" {
  name    = "my-account"
  profile = "default"

  cloud_native_protection {
    permission_groups = ["BASIC"]
    regions           = ["us-east-2"]
  }
}

data "polaris_object" "account" {
  name        = polaris_aws_account.account.name
  object_type = "AwsNativeAccount"
}

resource "polaris_refresh" "account" {
  object_id   = data.polaris_object.account.id
  object_type = "AwsNativeAccount"
  timestamp   = "2026-03-12T10:00:00Z"
}

data "polaris_object" "ec2" {
  name        = "my-instance"
  object_type = "AwsNativeEc2Instance"

  depends_on = [polaris_refresh.account]
}

# Azure example: wait for an Azure subscription to be refreshed after onboarding.
resource "polaris_azure_subscription" "sub" {
  subscription_id   = "00000000-0000-0000-0000-000000000000"
  subscription_name = "my-subscription"
  tenant_domain     = "my-tenant.onmicrosoft.com"

  cloud_native_protection {
    resource_group_name   = "my-resource-group"
    resource_group_region = "eastus2"

    regions = ["eastus2"]
  }
}

data "polaris_object" "sub" {
  name        = polaris_azure_subscription.sub.subscription_name
  object_type = "AzureNativeSubscription"
}

resource "polaris_refresh" "sub" {
  object_id   = data.polaris_object.sub.id
  object_type = "AzureNativeSubscription"
  timestamp   = "2026-03-12T10:00:00Z"

  timeouts {
    create = "90m"
  }
}

data "polaris_object" "vm" {
  name        = "my-vm"
  object_type = "AzureNativeVirtualMachine"

  depends_on = [polaris_refresh.sub]
}
