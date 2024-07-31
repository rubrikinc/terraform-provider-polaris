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
