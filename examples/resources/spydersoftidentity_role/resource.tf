resource "spydersoftidentity_role" "editor" {
  name = "editor"

  # Claims every member of the role receives. A type may repeat with different
  # values; the value defaults to an empty string. Types must not contain "/".
  claims = [
    { type = "permission", value = "articles.read" },
    { type = "permission", value = "articles.write" },
    { type = "urn:example:department", value = "content" },
  ]
}
