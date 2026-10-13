terraform {
  required_providers {
    spydersoftidentity = {
      source = "registry.terraform.io/spyder007/spydersoft-identity"
    }
  }
}

# Configured through the SPYDERSOFT_IDENTITY_* environment variables
# (see README -> Local development).
provider "spydersoftidentity" {}

resource "spydersoftidentity_scope" "example" {
  name         = "tf.example"
  display_name = "Terraform example scope"
  description  = "Created by the local provider smoke test"

  user_claims = ["email", "name"]

  properties = {
    owner = "terraform"
  }
}

output "scope_id" {
  value = spydersoftidentity_scope.example.id
}
