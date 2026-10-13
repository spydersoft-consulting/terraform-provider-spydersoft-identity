resource "spydersoftidentity_identity_resource" "example" {
  name         = "example-profile"
  display_name = "Example profile"

  user_claims = ["name", "nickname"]
}
