![Go version](https://img.shields.io/github/go-mod/go-version/rubrikinc/terraform-provider-polaris) ![License MIT](https://img.shields.io/github/license/rubrikinc/terraform-provider-polaris) ![Latest tag](https://img.shields.io/github/v/tag/rubrikinc/terraform-provider-polaris)

# Terraform Provider for Rubrik Polaris
For documentation of all resources and their parameters head over to the
[Terraform Registry Docs](https://registry.terraform.io/providers/rubrikinc/polaris/latest/docs). Note that the provider
requires Terraform version 0.15.x or newer.

## Use the Official Build
To use the official version of the provider built by Rubrik and published to the Terraform Registry, use the following
snippet at the top of your Terraform configuration:
```
terraform {
  required_providers {
    polaris = {
      source = "rubrikinc/polaris"
    }
  }
}
```
This will pull down the latest version of the provider from the Terraform Registry. Terraform will also validate the
authenticity of the provider using cryptographically signed checksums.

### Environment Variables
The following environmental variables can be used to override the default behaviour of the provider:
* *RUBRIK_POLARIS_LOGLEVEL* — Overrides the log level of the provider. Valid log levels are: *FATAL*, *ERROR*,
  *WARN*, *INFO*, *DEBUG*, *TRACE* and *OFF*. The default log level of the provider is *WARN*.
* *RUBRIK_POLARIS_TOKEN_CACHE* — Overrides whether the token cache should be used or not. By default, the token
  cache is used.
* *RUBRIK_POLARIS_TOKEN_CACHE_DIR* — Overrides the directory where cached authentication tokens are stored. By default,
  the OS default directory for temporary files are used.
* *RUBRIK_POLARIS_TOKEN_CACHE_SECRET* — Overrides the secret used as input when generating an encryption key for the
  authentication token.

### Provider Credentials
The provider supports both local user accounts and service accounts. For documentation on how to create either using
Polaris see the [Rubrik Support Portal](http://support.rubrik.com).

#### Local User Account
To use a local user account with the provider first create a directory called `.rubrik` in your home directory. In that
directory create a file called `polaris-accounts.json`. This JSON file can hold one or more local user accounts as per
this pattern:
```
{
  "<my-account>": {
    "username": "<my-username>",
    "password": "<my-password>",
    "url": "<my-polaris-url>",
  }
}
```
Where _my-account_ is an arbitrary name used to refer to the account when configuring the provider in the Terraform
configuration. _my-username_ and _my-password_ are the username and password of the local user account. _my-polaris-url_
is the URL of the Polaris API. The URL normally follows the pattern `https://{polaris-domain}.my.rubrik.com/api`. Which
is the same URL as for accessing the Polaris UI but with `/api` added to the end.

As an example, assume our Polaris domain is `my-polaris-domain` and that the username and password of our local user
account is `john.doe@example.org` and `password123` the content of the `polaris-accounts.json` file then should be:
```
{
  "johndoe": {
    "username": "john.doe@example.org",
    "password": "password123",
    "url": "https://my-polaris-domain.my.rubrik.com/api"
  }
}
```

Where `johndoe` will be used to refer to this account from our Terraform configuration:
```
terraform {
  required_providers {
    polaris = {
      source = "rubrikinc/polaris"
    }
  }
}

provider "polaris" {
  credentials = "johndoe"
}
```
##### Environment Variables for Local User Accounts
When using a local user account the following environmental variables can be used to override the default local user
account behaviour:
* *RUBRIK_POLARIS_ACCOUNT_CREDENTIALS* — Overrides the content of the local user account file.
* *RUBRIK_POLARIS_ACCOUNT_FILE* — Overrides the name and path of the file to read local user accounts from.
* *RUBRIK_POLARIS_ACCOUNT_NAME* — Overrides the name of the local user account given to the credentials
parameter in the provider configuration.
* *RUBRIK_POLARIS_ACCOUNT_USERNAME* — Overrides the username of the local user account.
* *RUBRIK_POLARIS_ACCOUNT_PASSWORD* — Overrides the password of the local user account.
* *RUBRIK_POLARIS_ACCOUNT_URL* — Overrides the Polaris API URL.

#### Service Account
To use a service account with the provider first download the service account credentials as a JSON file from the
Polaris User Management UI page. Next, configure the provider to use the the downloaded credentials file in the
Terraform configuration:
```
terraform {
  required_providers {
    polaris = {
      source = "rubrikinc/polaris"
    }
  }
}

provider "polaris" {
  credentials = "/path/to/credentials.json"
}
```
##### Environment Variables for Service Accounts
When using a service account the following environmental variables can be used to override the default service account
behaviour:
* *RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS* — Overrides the content of the service account credentials file.
* *RUBRIK_POLARIS_SERVICEACCOUNT_FILE* — Overrides the name and path of the service account credentials file.
* *RUBRIK_POLARIS_SERVICEACCOUNT_NAME* — Overrides the name of the service account.
* *RUBRIK_POLARIS_SERVICEACCOUNT_CLIENTID* — Overrides the client id of the service account.
* *RUBRIK_POLARIS_SERVICEACCOUNT_CLIENTSECRET* — Overrides the client secret of the service account.
* *RUBRIK_POLARIS_SERVICEACCOUNT_ACCESSTOKENURI* — Overrides the service account access token URI. When using a
service account the Polaris API URL is derived from this URI.

## Use Your Own Build
### Build
To build the provider for your machines OS and hardware architecture run the following command in the root of the
repository:
```
$ go build
```

After the build finishes you should have a binary named `terraform-provider-polaris` in the root of the repository.

### Install
To install the newly built provider on your machine you first need to create the directory where Terraform looks for
local providers:
```
$ mkdir -p ~/.terraform.d/plugins
```

Next you need to copy the provider binary into a subdirectory of `~/.terraform.d/plugins`, the exact subdirectory
depends on your machines OS and hardware architecture. For Linux/AMD64 the subdirectory would be
`terraform.rubrik.com/rubrikinc/polaris/0.0.1/linux_amd64`, where `0.0.1` is the version of the provider binary. This
can either be `0.0.1` or the release tag closest to the commit you built. For the release tag `v0.1.0` you would use
`0.1.0`. So the commands for copying a build of the `v0.1.0` release tag on a Linux/AMD64 machine would be:
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

### Use
To use the local provider in a Terraform configuration use the following snippet at the top of the file:
```
terraform {
  required_providers {
    polaris = {
      source  = "terraform.rubrik.com/rubrikinc/polaris"
    }
  }
}
```
