resource "spydersoftidentity_api_resource" "example" {
  name         = "example-api"
  display_name = "Example API"

  user_claims = ["role"]
  scopes      = [spydersoftidentity_scope.example.name]
  properties  = { owner = "platform" }
}
