resource "spydersoftidentity_client_secret" "example" {
  client      = spydersoftidentity_client.example.id
  value       = var.example_client_secret # sensitive; stored in state
  description = "managed by terraform"
  expiration  = "2027-12-31T00:00:00Z"
}
