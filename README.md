| :warning: Code in this repository is in BETA and should NOT be used in a production system! |
:---

# Terraform Provider for Rubrik Polaris

## Build
To build the provider for your machines OS and hardware architecture run the following command in the root of the repository:
```
$ go build
```

After the build finishes you should have a binary named `terraform-provider-polaris` in the root of the repository.

## Install
To install the newly built provider on your machine you first need to create the directory where Terraform looks for local providers:
```
$ mkdir -p ~/.terraform.d/plugins
```

Next you need to copy the provider binary into a subdirectory of `~/.terraform.d/plugins`, the exact subdirectory depends on your machines OS and hardware architecture. For Linux/AMD64 the subdirectory would be `terraform.rubrik.com/rubrikinc/polaris/0.0.1/linux_amd64`, where `0.0.1` is the version of the provider binary. This can either be `0.0.1` or the release tag closest to the commit you built. For the release tag `v0.1.0` you would use `0.1.0`. So the commands for copying a build of the `v0.1.0` release tag on a Linux/AMD64 machine would be:
```
$ mkdir ~/.terraform.d/plugins/terraform.rubrik.com/rubrikinc/polaris/0.1.0/linux_amd64
$ cp terraform-provider-polaris ~/.terraform.d/plugins/terraform.rubrik.com/rubrikinc/polaris/0.1.0/linux_amd64
```

After the above commands the directory structure under `~/.terraform.d` would be:
```
.terraform.d/
└── plugins/
    └── terraform.rubrik.com/
        └── rubrik/
            └── polaris/
                └── 0.1.0/
                    └── linux_amd64/
                        └── terraform-provider-polaris
```

## Use
When using the installed local provider in a Terraform configuration file you need to explicitly tell Terraform about it using the following snippet at the top of the file:
```
terraform {
  required_providers {
    polaris = {
      source  = "terraform.rubrik.com/rubrikinc/polaris"
    }
  }
}
```

Note that the provider requires Terraform version 0.15.x or newer.

For details on how to configure the provider or how to use different resources have a look in the `docs/` and `examples/` subdirectories.
