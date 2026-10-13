terraform {
  required_providers {
    spydersoftidentity = {
      source = "registry.terraform.io/spydersoft-consulting/spydersoft-identity"
    }
  }
}

provider "spydersoftidentity" {}

variable "secret_value" {
  type      = string
  sensitive = true
}

resource "spydersoftidentity_scope" "api" {
  name         = "tf-test.api"
  display_name = "Terraform test API scope"
  user_claims  = ["email"]
}

resource "spydersoftidentity_identity_resource" "profile" {
  name         = "tf-test.profile"
  display_name = "Terraform test identity resource"
  user_claims  = ["name", "nickname"]
  properties   = { owner = "terraform" }
}

resource "spydersoftidentity_api_resource" "api" {
  name         = "tf-test-api"
  display_name = "Terraform test API"
  user_claims  = ["role"]
  scopes       = [spydersoftidentity_scope.api.name]
  properties   = { owner = "terraform" }
}

resource "spydersoftidentity_client" "app" {
  client_id   = "tf-test-client"
  client_name = "Terraform test client"
  enabled     = false

  require_consent = false
  require_pkce    = false

  allowed_grant_types = ["client_credentials"]
  allowed_scopes      = [spydersoftidentity_scope.api.name]
  redirect_uris       = ["https://localhost/cb"]
  allowed_cors_origins = ["https://localhost"]

  claims = [
    { type = "role", value = "a" },
    { type = "role", value = "b" },
  ]
  properties = { owner = "terraform" }
}

resource "spydersoftidentity_client_secret" "app" {
  client      = spydersoftidentity_client.app.id
  value       = var.secret_value
  description = "terraform test"
}

resource "spydersoftidentity_api_resource_secret" "api" {
  api_resource = spydersoftidentity_api_resource.api.id
  value        = var.secret_value
  description  = "terraform test"
}
