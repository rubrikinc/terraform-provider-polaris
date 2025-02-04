---
page_title: "Provider: RSC"
---

# RSC Provider
The RSC provider, formerly known as the Polaris provider, provides resources to interact with the Rubrik RSC platform.
Additional examples on how to use the provider are available in the
[terraform-provider-polaris-examples](https://github.com/rubrikinc/terraform-provider-polaris-examples) GitHub
repository. Documentation for the Rubrik Security Cloud is available at https://docs.rubrik.com/en-us/saas/index.html.

!> Since v0.7.0, all RSC authentication tokens are cached on disk by default. This default behavior can be turned off
   by setting the `token_cache` provider configuration field to `false` or the `RUBRIK_POLARIS_TOKEN_CACHE` environment
   variable to `FALSE`.

## Configuration

### Authentication Token Cache
All RSC authentication tokens are cached on disk by default. They are placed in the operating system's default
directory for temporary files (`$TMPDIR` or `/tmp` on Linux and `%TMP%`, `%TEMP%` or `%USERPROFILE%` on Windows), this
can be overridden using the `token_cache_dir` provider configuration field or the `RUBRIK_POLARIS_TOKEN_CACHE_DIR`
environmental variable.

Each authentication token written to the cache is encrypted using 256-bit AES encryption. The encryption key is derived
from the RSC account information passed to the provider, this can be overridden using the `token_cache_secret` provider
configuration field or the `RUBRIK_POLARIS_TOKEN_CACHE_SECRET` environmental variable. When a secret is provided, the
encryption key will be derived from the secret instead of the account information.

The cache can be disabled by setting the `token_cache` provider configuration field to `false` or the
`RUBRIK_POLARIS_TOKEN_CACHE` environmental variable to `FALSE`.

### Service Account
First download the service account credentials as a JSON file from the RSC User Management UI page. Next, configure the
provider to use the downloaded credentials file in the Terraform configuration:
```terraform
provider "polaris" {
  credentials = "/path/to/service-account-credentials.json"
}
```
The content of the service account file, the JSON data, can also be passed directly to the provider configuration:
```terraform
provider "polaris" {
  credentials = "<content of service-account-credentials.json>"
}
```
The service account can also be passed to the provider using one of the `RUBRIK_POLARIS_SERVICEACCOUNT_FILE` and
`RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS` environment variables. The `RUBRIK_POLARIS_SERVICEACCOUNT_FILE` environment
variable should contain the path to the downloaded credentials file. The `RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS`
environment variable should contain the JSON data content of the downloaded credentials file. When passing the service
account using the environment variable, leave the provider configuration empty:
```terraform
provider "polaris" {}
```
For documentation on how to create a service account using RSC, visit the
[Rubrik Support Portal](http://support.rubrik.com).

### Local User Account
First create a directory called `.rubrik` in your home directory. Next, create a file called `polaris-accounts.json` in
that directory. This JSON file holds one or more local user accounts:
```json
{
  "&lt;my-account&gt;": {
    "username": "&lt;my-username&gt;",
    "password": "&lt;my-password&gt;",
    "url": "&lt;my-rsc-url&gt;"
  }
}
```
Where *my-account* is an arbitrary name used to refer to the account when configuring the provider. *my-username* and
*my-password* are the username and password of the local user account. *my-rsc-url* is the URL of the RSC API. The
URL follows the pattern `https://{rsc-domain}.my.rubrik.com/api`. Which is the same URL as for accessing the RSC UI but
with `/api` added to the end.

To configure the provider to use a local user account specify the name as the provider credentials:
```terraform
provider "polaris" {
  credentials = "my-account"
}
```
For documentation on how to create a local user account using RSC, visit the
[Rubrik Support Portal](http://support.rubrik.com).

### Environment Variables
The following environmental variables can be used to override the default behavior of the provider:
* `RUBRIK_POLARIS_LOGLEVEL` - Overrides the log level of the provider. Valid log levels are: `FATAL`, `ERROR`, `WARN`,
  `INFO`, `DEBUG`, `TRACE` and `OFF`. The default log level of the provider is `WARN`.
* `RUBRIK_POLARIS_TOKEN_CACHE` - Overrides whether the token cache should be used or not. By default, the token
  cache is used.
* `RUBRIK_POLARIS_TOKEN_CACHE_DIR` - Overrides the directory where cached authentication tokens are stored. By
  default, the OS default directory for temporary files is used.
* `RUBRIK_POLARIS_TOKEN_CACHE_SECRET` - Overrides the secret used as input when generating an encryption key for the
  authentication token.

When using a service account the following environmental variables can be used to override the default service account
behavior:
* `RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS` - Overrides the content of the service account credentials file.
* `RUBRIK_POLARIS_SERVICEACCOUNT_FILE` - Overrides the name and path of the service account credentials file.
* `RUBRIK_POLARIS_SERVICEACCOUNT_NAME` - Overrides the name of the service account.
* `RUBRIK_POLARIS_SERVICEACCOUNT_CLIENTID` - Overrides the client id of the service account.
* `RUBRIK_POLARIS_SERVICEACCOUNT_CLIENTSECRET` - Overrides the client secret of the service account.
* `RUBRIK_POLARIS_SERVICEACCOUNT_ACCESSTOKENURI` - Overrides the service account access token URI. When using a service
  account the RSC API URL is derived from this URI.

When using a local user account the following environmental variables can be used to override the default local user
account behavior:
* `RUBRIK_POLARIS_ACCOUNT_CREDENTIALS` - Overrides the content of the local user accounts file.
* `RUBRIK_POLARIS_ACCOUNT_FILE` - Overrides the name and path of the file to read local user accounts from.
* `RUBRIK_POLARIS_ACCOUNT_NAME` - Overrides the name of the local user account given to the credentials parameter in the
  provider configuration.
* `RUBRIK_POLARIS_ACCOUNT_USERNAME` - Overrides the username of the local user account.
* `RUBRIK_POLARIS_ACCOUNT_PASSWORD` - Overrides the password of the local user account.
* `RUBRIK_POLARIS_ACCOUNT_URL` - Overrides the RSC API URL.

## Example Usage

```terraform
# Service account from the current environment.
provider "polaris" {
}

# Service account from the content of a service account file.
provider "polaris" {
  credentials = <<-EOS
    {
      "client_id": "client|...",
      "client_secret": "...",
      "name": "dummy-service-account",
      "access_token_uri": "https://account.my.rubrik.com/api/client_token"
    }
    EOS
}

provider "polaris" {
  credentials = "<content of service-account-credentials.json>"
}

# Service account from file.
provider "polaris" {
  credentials = "/path/to/service-account-credentials.json"
}

# Local user account.
provider "polaris" {
  credentials = "my-account"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `credentials` (String) The service account credentials, service account credentials file name or local user account name to use when accessing RSC.
- `token_cache` (Boolean) Enable or disable the token cache. The token cache is enabled by default.
- `token_cache_dir` (String) The directory where cached authentication tokens are stored. The OS directory for temporary files is used by default.
- `token_cache_secret` (String, Sensitive) The secret used as input when generating an encryption key for the authentication token. The encryption key is derived from the RSC account information by default.
