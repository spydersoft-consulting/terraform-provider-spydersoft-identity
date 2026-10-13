resource "spydersoftidentity_scope" "example" {
  name         = "example:read"
  display_name = "Read example data"
  description  = "Allows reading example data."

  user_claims = ["email"]
  properties  = { owner = "platform" }
}
