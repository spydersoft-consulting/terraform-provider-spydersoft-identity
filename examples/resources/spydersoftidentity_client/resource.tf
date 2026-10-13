resource "spydersoftidentity_client" "example" {
  client_id   = "example-web"
  client_name = "Example web app"

  allowed_grant_types = ["authorization_code"]
  allowed_scopes      = ["openid", "profile", spydersoftidentity_scope.example.name]
  redirect_uris       = ["https://app.example.com/signin-oidc"]

  post_logout_redirect_uris = ["https://app.example.com/signout-callback-oidc"]
  allowed_cors_origins      = ["https://app.example.com"]
  allow_offline_access      = true

  # Claims may repeat a type with different values.
  claims = [
    { type = "role", value = "reader" },
    { type = "role", value = "writer" },
  ]
}
