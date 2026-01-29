---
page_title: "polaris_aws_exocompute Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_aws_exocompute` resource creates an RSC Exocompute configuration
for AWS workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When an RSC managed host configuration is created, RSC
    will automatically deploy the necessary resources in the specified AWS
    region to run the Exocompute service. AWS security groups can be managed by
    RSC or by the customer.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the AWS EKS
    cluster attached by the customer, using the
    `polaris_aws_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Items 1 and 2 above requires that the AWS account has been onboarded with the
`EXOCOMPUTE` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_aws_exocompute` resource:
 1. Using the `account_id`, `region`, `vpc_id` and `subnets` fields creates an
    RSC managed host configuration. The `cluster_security_group_id` and
    `node_security_group_id` fields can be used to create an Exocompute
    configuration where the customer manage the security groups.
 2. Using the `account_id` and `region` fields creates a customer managed host
    configuration. Note, the `polaris_aws_exocompute_cluster_attachment`
    resource must be used to attach an AWS EKS cluster to the Exocompute
    configuration.
 3. Using the `account_id` and `host_cloud_account_id` fields creates an
    application configuration.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.

---

# polaris_aws_exocompute (Resource)


The `polaris_aws_exocompute` resource creates an RSC Exocompute configuration
for AWS workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When an RSC managed host configuration is created, RSC
    will automatically deploy the necessary resources in the specified AWS
    region to run the Exocompute service. AWS security groups can be managed by
    RSC or by the customer.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the AWS EKS
    cluster attached by the customer, using the
    `polaris_aws_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Items 1 and 2 above requires that the AWS account has been onboarded with the
`EXOCOMPUTE` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_aws_exocompute` resource:
 1. Using the `account_id`, `region`, `vpc_id` and `subnets` fields creates an
    RSC managed host configuration. The `cluster_security_group_id` and
    `node_security_group_id` fields can be used to create an Exocompute
    configuration where the customer manage the security groups.
 2. Using the `account_id` and `region` fields creates a customer managed host
    configuration. Note, the `polaris_aws_exocompute_cluster_attachment`
    resource must be used to attach an AWS EKS cluster to the Exocompute
    configuration.
 3. Using the `account_id` and `host_cloud_account_id` fields creates an
    application configuration.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.



## Example Usage

```terraform
data "polaris_aws_account" "host" {
  name = "host-account"
}

# RSC managed Exocompute and security groups.
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# RSC managed Exocompute with private cluster access.
resource "polaris_aws_exocompute" "host_private" {
  account_id     = data.polaris_aws_account.host.id
  region         = "us-east-2"
  vpc_id         = "vpc-4859acb9"
  cluster_access = "EKS_CLUSTER_ACCESS_TYPE_PRIVATE"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# RSC managed Exocompute and customer managed security groups.
resource "polaris_aws_exocompute" "host" {
  account_id                = data.polaris_aws_account.host.id
  cluster_security_group_id = "sg-005656347687b8170"
  node_security_group_id    = "sg-00e147656785d7e2f"
  region                    = "us-east-2"
  vpc_id                    = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# Customer managed Exocompute.
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
}

resource "polaris_aws_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-eks-cluster"
  exocompute_id = polaris_aws_exocompute.host.id
}

data "polaris_aws_account" "application" {
  name = "application-account"
}

# Application Exocompute.
resource "polaris_aws_exocompute" "application" {
  account_id      = data.polaris_aws_account.application.id
  host_account_id = data.polaris_aws_account.host.id
}
```


## Schema

### Required

- `account_id` (String) RSC cloud account ID (UUID). Changing this forces a new resource to be created.

### Optional

- `cluster_access` (String) EKS cluster access type. Possible values are `EKS_CLUSTER_ACCESS_TYPE_PUBLIC` and `EKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Defaults to `EKS_CLUSTER_ACCESS_TYPE_PUBLIC`. Can only be used with RSC managed configurations.
- `cluster_security_group_id` (String) AWS security group ID for the cluster. Changing this forces a new resource to be created.
- `host_account_id` (String) Exocompute host cloud account ID. Changing this forces a new resource to be created.
- `node_security_group_id` (String) AWS security group ID for the nodes. Changing this forces a new resource to be created.
- `region` (String) AWS region to run the Exocompute instance in. Changing this forces a new resource to be created.
- `subnets` (Set of String) AWS subnet IDs for the cluster subnets. Changing this forces a new resource to be created.
- `vpc_id` (String) AWS VPC ID for the cluster network. Changing this forces a new resource to be created.

### Read-Only

- `id` (String) Exocompute configuration ID (UUID).
- `polaris_managed` (Boolean) If true the security groups are managed by RSC.

## Import

To import an application exocompute configuration prepend `app-` to the ID of the configuration.

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_aws_exocompute.host
  id = "58e2a8bb-078d-4f67-8b66-5515fd701c8e"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_aws_exocompute.host 58e2a8bb-078d-4f67-8b66-5515fd701c8e
```

