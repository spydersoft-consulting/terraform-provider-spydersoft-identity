resource "spydersoftidentity_api_resource_secret" "example" {
  api_resource = spydersoftidentity_api_resource.example.id
  value        = var.example_api_secret # sensitive; stored in state
  description  = "managed by terraform"
}
