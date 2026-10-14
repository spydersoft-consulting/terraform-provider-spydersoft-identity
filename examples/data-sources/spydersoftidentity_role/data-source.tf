data "spydersoftidentity_role" "admin" {
  name = "admin"
}

output "admin_role_id" {
  value = data.spydersoftidentity_role.admin.id
}
