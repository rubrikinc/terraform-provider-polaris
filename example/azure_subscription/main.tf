#  Copyright 2021 Rubrik, Inc.
#
#  Permission is hereby granted, free of charge, to any person obtaining a copy
#  of this software and associated documentation files (the "Software"), to
#  deal in the Software without restriction, including without limitation the
#  rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
#  sell copies of the Software, and to permit persons to whom the Software is
#  furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included in
#  all copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
#  FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
#  DEALINGS IN THE SOFTWARE.

# Terraform configuration. Points Terraform to the Polaris provider.
terraform {
  required_providers {
    polaris = {
      source  = "terraform.rubrik.com/rubrik/polaris"
    }
  }
}

# Polaris provider configuration. Points the provider to the Polaris service
# account to use.
provider "polaris" {
  credentials = "${path.module}/polaris-service-account.json"
}

# Resource configuration. Add the Azure service principal to Polaris.
resource "polaris_azure_service_principal" "default" {
    credentials = "${path.module}/service-principal.json"
}

# Resource configuration. Add the Azure subscription to Polaris.
resource "polaris_azure_subscription" "default" {
    subscription_id   = "8fa81a5e-a236-4a73-8e28-e1dcf863c56d"
    subscription_name = "Trinity-FDSE"
    tenant_domain     = "rubriktrinity.onmicrosoft.com"
    regions           = ["eastus2"]

    depends_on = [polaris_azure_service_principal.default]
}
