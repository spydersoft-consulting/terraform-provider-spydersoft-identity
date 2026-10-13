package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccClientConfig(scope, clientID, redirects, claims, secret string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "spydersoftidentity_scope" "test" {
  name = %[1]q
}

resource "spydersoftidentity_client" "test" {
  client_id   = %[2]q
  client_name = "Acceptance test client"
  enabled     = false

  require_consent = false
  require_pkce    = false

  allowed_grant_types  = ["client_credentials"]
  allowed_scopes       = [spydersoftidentity_scope.test.name]
  redirect_uris        = %[3]s
  allowed_cors_origins = ["https://localhost"]
  claims               = %[4]s
  properties           = { owner = "acc" }
}

resource "spydersoftidentity_client_secret" "test" {
  client      = spydersoftidentity_client.test.id
  value       = %[5]q
  description = "acceptance test"
}

data "spydersoftidentity_client" "test" {
  client_id = spydersoftidentity_client.test.client_id
}
`, scope, clientID, redirects, claims, secret)
}

func TestAccClient(t *testing.T) {
	scope := testAccName("clscope")
	clientID := testAccName("client")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientConfig(scope, clientID,
					`["https://localhost/cb"]`,
					`[{ type = "role", value = "a" }, { type = "role", value = "b" }]`,
					"acceptance-test-secret-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "client_id", clientID),
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "protocol_type", "oidc"),
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "allowed_grant_types.#", "1"),
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "redirect_uris.#", "1"),
					// Two claims sharing a type must both survive.
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "claims.#", "2"),
					resource.TestCheckResourceAttrSet("spydersoftidentity_client_secret.test", "id"),
					resource.TestCheckResourceAttrPair("data.spydersoftidentity_client.test", "id", "spydersoftidentity_client.test", "id"),
					resource.TestCheckResourceAttr("data.spydersoftidentity_client.test", "claims.#", "2"),
				),
			},
			{
				// In-place client update plus a secret replacement.
				Config: testAccClientConfig(scope, clientID,
					`["https://localhost/cb", "https://localhost/cb2"]`,
					`[{ type = "role", value = "a" }, { type = "dept", value = "x" }]`,
					"acceptance-test-secret-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("spydersoftidentity_client.test", "claims.#", "2"),
				),
			},
			{
				ResourceName:      "spydersoftidentity_client.test",
				ImportState:       true,
				ImportStateId:     clientID,
				ImportStateVerify: true,
			},
		},
	})
}
