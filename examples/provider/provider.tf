terraform {
  required_providers {
    spydersoftidentity = {
      source = "spydersoft-consulting/spydersoft-identity"
    }
  }
}

# All four settings can instead come from the SPYDERSOFT_IDENTITY_URL,
# SPYDERSOFT_IDENTITY_TOKEN_URL, SPYDERSOFT_IDENTITY_CLIENT_ID and
# SPYDERSOFT_IDENTITY_CLIENT_SECRET environment variables.
provider "spydersoftidentity" {
  url           = "https://auth-admin-api.example.com"
  token_url     = "https://auth.example.com/connect/token"
  client_id     = "terraform-provider"
  client_secret = var.identity_client_secret
}
