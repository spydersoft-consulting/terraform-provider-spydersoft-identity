# terraform-provider-spydersoft-identity

Terraform / OpenTofu provider for the Spydersoft Identity server admin API
(`identity_server` → `Spydersoft.Identity.Admin.Api`).

> **Status: early.** Only `spydersoftidentity_scope` exists so far; it has been
> verified end to end (create, update, import, destroy) against a live server.

## Planned resources

| Resource | Status |
| --- | --- |
| `spydersoftidentity_scope` (incl. claims, properties) | modelled |
| `spydersoftidentity_api_resource` | todo |
| `spydersoftidentity_identity_resource` | todo |
| `spydersoftidentity_client` | todo |
| client / API resource secrets (write-only secret handling) | todo |

Sub-resources (redirect URIs, scopes, claims, ...) are value-keyed on the server
and reject duplicates with `409 Conflict`, so they can be modelled as set
attributes on the parent resource.

## Provider configuration

```hcl
provider "spydersoftidentity" {
  url           = "https://identity-admin.example.com"         # SPYDERSOFT_IDENTITY_URL
  token_url     = "https://identity.example.com/connect/token" # SPYDERSOFT_IDENTITY_TOKEN_URL
  client_id     = "terraform-provider"                         # SPYDERSOFT_IDENTITY_CLIENT_ID
  client_secret = var.identity_client_secret                   # SPYDERSOFT_IDENTITY_CLIENT_SECRET
}

terraform {
  required_providers {
    spydersoftidentity = { source = "registry.terraform.io/spyder007/spydersoft-identity" }
  }
}

resource "spydersoftidentity_scope" "example" {
  name         = "example:read"
  display_name = "Read example data"

  user_claims = ["email"]
  properties  = { owner = "platform" }
}
```

The client is a `client_credentials` client holding `identity:admin:read` and
`identity:admin:write`. A `terraform-provider` client is seeded for local
development by the identity server's data seeder.

## Local development

One-time setup (Windows):

```powershell
winget install GoLang.Go          # then open a new shell
go mod tidy                       # generates go.sum
```

**1. Unit tests** (no server needed; the client is tested against a fake HTTP API):

```powershell
go vet ./...
go test ./...
```

**2. Run the identity server locally.** From the `identity_server` repo, on a
branch that has the `by-name` endpoints and the seeded `terraform-provider`
client (PR 25 or later):

```powershell
aspire run        # or: dotnet run --project source/Spydersoft.Identity.AppHost
```

In the Aspire dashboard, start the `seeder` resource once (it is
explicit-start in dev). Endpoints: identity `http://localhost:7020`,
admin API `http://localhost:7030`.

**3. Build and point OpenTofu/Terraform at your local build:**

```powershell
go install .
Copy-Item examples/local/dev.tfrc.example examples/local/dev.tfrc   # edit the path
$env:TF_CLI_CONFIG_FILE = "$PWD\examples\local\dev.tfrc"
$env:SPYDERSOFT_IDENTITY_URL           = "http://localhost:7030"
$env:SPYDERSOFT_IDENTITY_TOKEN_URL     = "http://localhost:7020/connect/token"
$env:SPYDERSOFT_IDENTITY_CLIENT_ID     = "terraform-provider"
$env:SPYDERSOFT_IDENTITY_CLIENT_SECRET = "local-dev-only-do-not-deploy"

cd examples/local
tofu plan          # no `init` needed with dev_overrides
tofu apply
tofu plan          # should report no changes (idempotency check)
tofu import spydersoftidentity_scope.example tf.example   # after removing it from state
tofu destroy
```

Re-run `go install .` after every provider change; `dev_overrides` uses the
installed binary.

## Import

```sh
terraform import spydersoftidentity_scope.example example:read
```

Import uses the natural key (scope name) via `GET /api/v1/Scopes/by-name/{name}`.

## Notes

- The module path and registry address (`registry.terraform.io/spyder007/spydersoft-identity`)
  are placeholders until the org rename settles; changing them is a find/replace
  in `go.mod`, `main.go` and the imports.
- Avoid publishing a release to the public registry before the name is final.
