data "spydersoftidentity_scope" "email" {
  name = "email"
}

output "email_scope_claims" {
  value = data.spydersoftidentity_scope.email.user_claims
}
