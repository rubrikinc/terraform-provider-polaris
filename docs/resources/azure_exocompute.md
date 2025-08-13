---
page_title: "polaris_azure_exocompute Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration
for Azure workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When a host configuration is created, RSC will
    automatically deploy the necessary resources in the specified Azure region
    to run the Exocompute service. A host configuration can be used by both the
    host cloud account and application cloud accounts mapped to the host
    account.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the Azure
    AKS cluster attached by the customer, using the
    `polaris_azure_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Item 1 and 2 above requires that the Azure subscription has been onboarded with
the `exocompute` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_azure_exocompute` resource:
 1. Using the `cloud_account_id`, `region`, `subnet` and
   `pod_overlay_network_cidr` fields creates an RSC managed host configuration.
 2. Using the `cloud_account_id` and `region` fields creates a customer managed
    host configuration. Note, the `polaris_azure_exocompute_cluster_attachment`
    resource must be used to attach an Azure AKS cluster to the Exocompute
    configuration.
 3. Using the `cloud_account_id` and `host_cloud_account_id` fields creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   `pod_overlay_network_cidr` field, this is discouraged and should only be done
   for backwards compatibility reasons.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.

---

# polaris_azure_exocompute (Resource)


The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration
for Azure workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When a host configuration is created, RSC will
    automatically deploy the necessary resources in the specified Azure region
    to run the Exocompute service. A host configuration can be used by both the
    host cloud account and application cloud accounts mapped to the host
    account.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the Azure
    AKS cluster attached by the customer, using the
    `polaris_azure_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Item 1 and 2 above requires that the Azure subscription has been onboarded with
the `exocompute` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_azure_exocompute` resource:
 1. Using the `cloud_account_id`, `region`, `subnet` and
   `pod_overlay_network_cidr` fields creates an RSC managed host configuration.
 2. Using the `cloud_account_id` and `region` fields creates a customer managed
    host configuration. Note, the `polaris_azure_exocompute_cluster_attachment`
    resource must be used to attach an Azure AKS cluster to the Exocompute
    configuration.
 3. Using the `cloud_account_id` and `host_cloud_account_id` fields creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   `pod_overlay_network_cidr` field, this is discouraged and should only be done
   for backwards compatibility reasons.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.



## Example Usage

```terraform
data "polaris_azure_subscription" "host" {
  name = "host-subscription"
}

# RSC managed Exocompute.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id         = data.polaris_azure_subscription.host.id
  pod_overlay_network_cidr = "10.244.0.0/16"
  region                   = "eastus2"
  subnet                   = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test/subnets/default"
}

# Customer managed Exocompute.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id = data.polaris_azure_subscription.host.id
  region           = "eastus2"
}

resource "polaris_azure_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-aks-cluster"
  exocompute_id = polaris_azure_exocompute.host.id
}


data "polaris_azure_subscription" "application" {
  name = "application-subscription"
}

# Application Exocompute.
resource "polaris_azure_exocompute" "application" {
  cloud_account_id      = data.polaris_azure_subscription.application.id
  host_cloud_account_id = data.polaris_azure_subscription.host.id
}
```


## Schema

### Optional

- `cloud_account_id` (String) RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for which the Exocompute service runs. Changing this forces a new resource to be created.
- `host_cloud_account_id` (String) RSC cloud account ID of the shared exocompute host account. Changing this forces a new resource to be created.
- `pod_overlay_network_cidr` (String) The CIDR range assigned to pods when launching Exocompute with the CNI overlay network plugin mode. Changing this forces a new resource to be created.
- `region` (String) Azure region to run the exocompute service in. Should be specified in the standard Azure style, e.g. `eastus`. Changing this forces a new resource to be created.
- `subnet` (String) Azure subnet ID of the cluster subnet corresponding to the Exocompute configuration. This subnet will be used to allocate IP addresses to the nodes of the cluster. Changing this forces a new resource to be created.
- `subscription_id` (String, Deprecated) RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for which the Exocompute service runs. Changing this forces a new resource to be created. **Deprecated:** use `cloud_account_id` instead.

### Read-Only

- `id` (String) Exocompute configuration ID (UUID).

## Import

To import an application exocompute configuration prepend `app-` to the ID of the configuration.

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_azure_exocompute.host
  id = "a9caddfd-25bd-4327-85f6-fa698ed898b6"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_azure_exocompute.host a9caddfd-25bd-4327-85f6-fa698ed898b6
```

