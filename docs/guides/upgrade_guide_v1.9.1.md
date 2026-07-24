---
page_title: "Upgrade Guide: v1.9.1"
---

# Upgrade Guide v1.9.1

The v1.9.1 release adds support for onboarding and reading Azure DevOps organizations, projects and repositories, and
for looking up the permissions RSC requires for an Azure DevOps feature. See the [changelog](changelog.md) for the full
list of changes.

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

~> **Note:** If you are upgrading across multiple minor versions, review the upgrade guide for each intermediate
version as well. Each guide documents breaking changes and migration steps specific to that release.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.9.1 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.9.1"
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
The v1.9.1 release only adds new resources and data sources, so existing configurations are unaffected and no changes
are required. Proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.9.1 version.

## New Features

### Azure DevOps Onboarding

A new `polaris_azure_devops_organization` resource onboards an Azure DevOps organization to RSC using a
customer-supplied application (non-OAuth). Onboarding has three steps that map to three Terraform objects:

1. Register the customer application for the Azure DevOps use case with a `polaris_azure_service_principal` resource,
   setting the new `use_case = "AZURE_DEVOPS"` field.
2. Generate the onboarding scripts with the `polaris_azure_devops_script` data source and run them against the
   organization out of band. The provider does not run the scripts — run each one with the Azure CLI signed in
   (`az login`) as a Project Collection Administrator in the organization; the script mints a short-lived Azure DevOps
   token from that session, so no personal access token is required.
3. Onboard the organization with the `polaris_azure_devops_organization` resource.

```terraform
resource "polaris_azure_service_principal" "devops" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My DevOps App"
  app_secret    = "<my-apps-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
  use_case      = "AZURE_DEVOPS"
}

data "polaris_azure_devops_permissions" "repo" {
  feature           = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
  permission_groups = ["BASIC", "RECOVERY"]
}

data "polaris_azure_devops_script" "onboard" {
  org_native_ids = ["my-org"]
  tenant_domain  = polaris_azure_service_principal.devops.tenant_domain

  feature {
    name              = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
  }
}

resource "polaris_azure_devops_organization" "org" {
  native_id            = "my-org"
  tenant_domain        = polaris_azure_service_principal.devops.tenant_domain
  exocompute_host_type = "RUBRIK_HOST"
  exocompute_region    = "eastus"
  storage_type         = "RCV"

  feature {
    name              = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
    permissions       = data.polaris_azure_devops_permissions.repo.id
  }

  depends_on = [polaris_azure_service_principal.devops]
}
```

The `use_case` field on `polaris_azure_service_principal` selects whether the application is registered for cloud
native protection (the default) or Azure DevOps. Credentials are stored separately per use case, so a tenant that uses
both declares one service principal per use case. Omitting the field preserves the existing cloud native protection
behavior, so existing service principal configurations are unaffected.

### Updating Permissions

The permissions RSC requires for an Azure DevOps feature can change over time. The `polaris_azure_devops_permissions`
data source returns the current permissions for a feature and its permission groups, together with a
`permission_group_versions` map and an `id` that is a hash of the feature, permissions and versions. The `id` changes
whenever RSC updates the required permissions.

Wire the data source's `id` into the `permissions` field of the matching `feature` block on the
`polaris_azure_devops_organization` resource, as shown above. When RSC changes the required permissions, the `id`
changes and Terraform plans an update to the organization. Before applying, re-run the onboarding script against the
organization (see the `polaris_azure_devops_script` data source) to grant the new permissions; applying then notifies
RSC that they have been granted.

The `permissions` field is optional — omit it to manage the feature's permission groups without tracking permission
version changes.

### Reading Azure DevOps Objects

Three new data sources read onboarded Azure DevOps objects by RSC ID: `polaris_azure_devops_organization`,
`polaris_azure_devops_project` and `polaris_azure_devops_repository`.

The `polaris_object` data source also gains support for the `AzureDevOpsOrganization`, `AzureDevOpsProject` and
`AzureDevOpsRepository` object types, resolving an object to its RSC ID by name for use with the
`polaris_sla_domain_assignment` resource. Because project and repository names are only unique within their parent, set
the optional `org_id` (for a project) or `project_id` (for a repository) to disambiguate a name shared across parents:

```terraform
data "polaris_object" "repo" {
  object_type = "AzureDevOpsRepository"
  name        = "my-repo"
  project_id  = data.polaris_object.project.id
}
```

### Discovery and Import

A new `polaris_azure_devops_organization` list resource lists onboarded Azure DevOps organizations. Declare it in a
`.tfquery.hcl` file:

```terraform
list "polaris_azure_devops_organization" "all" {
  provider = polaris
}
```

Run `terraform query` to discover organizations, or `terraform query -generate-config-out=generated.tf` to also generate
a `resource` block and a matching `import` block for each one. RSC does not return the `cloud` type or the enabled
`feature` blocks, so the generated configuration imports every organization as `PUBLIC` with no features. Before
applying, edit `generated.tf`: set `cloud` in each generated import identity to `CHINA` or `USGOV` for any non-public
organization, and add at least one `feature` block to each resource.
