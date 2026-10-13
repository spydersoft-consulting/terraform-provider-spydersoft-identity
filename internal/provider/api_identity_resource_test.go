package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccApiResourceConfig(scope, name, claims string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "spydersoftidentity_scope" "test" {
  name = %[1]q
}

resource "spydersoftidentity_api_resource" "test" {
  name         = %[2]q
  display_name = "Acceptance test API"
  user_claims  = %[3]s
  scopes       = [spydersoftidentity_scope.test.name]
  properties   = { owner = "acc" }
}

resource "spydersoftidentity_api_resource_secret" "test" {
  api_resource = spydersoftidentity_api_resource.test.id
  value        = "acceptance-test-secret-value"
  description  = "acceptance test"
}

data "spydersoftidentity_api_resource" "test" {
  name = spydersoftidentity_api_resource.test.name
}
`, scope, name, claims)
}

func TestAccApiResource(t *testing.T) {
	scope := testAccName("apiscope")
	name := testAccName("api")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiResourceConfig(scope, name, `["role"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_api_resource.test", "name", name),
					resource.TestCheckResourceAttr("spydersoftidentity_api_resource.test", "user_claims.#", "1"),
					resource.TestCheckResourceAttr("spydersoftidentity_api_resource.test", "scopes.#", "1"),
					resource.TestCheckResourceAttrSet("spydersoftidentity_api_resource_secret.test", "id"),
					resource.TestCheckResourceAttrPair("data.spydersoftidentity_api_resource.test", "id", "spydersoftidentity_api_resource.test", "id"),
					resource.TestCheckResourceAttr("data.spydersoftidentity_api_resource.test", "scopes.#", "1"),
				),
			},
			{
				Config: testAccApiResourceConfig(scope, name, `["role", "email"]`),
				Check:  resource.TestCheckResourceAttr("spydersoftidentity_api_resource.test", "user_claims.#", "2"),
			},
			{
				ResourceName:      "spydersoftidentity_api_resource.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccIdentityResourceConfig(name, claims string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "spydersoftidentity_identity_resource" "test" {
  name         = %[1]q
  display_name = "Acceptance test identity resource"
  user_claims  = %[2]s
  properties   = { owner = "acc" }
}

data "spydersoftidentity_identity_resource" "test" {
  name = spydersoftidentity_identity_resource.test.name
}
`, name, claims)
}

func TestAccIdentityResource(t *testing.T) {
	name := testAccName("idres")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIdentityResourceConfig(name, `["name"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_identity_resource.test", "user_claims.#", "1"),
					resource.TestCheckResourceAttr("spydersoftidentity_identity_resource.test", "show_in_discovery_document", "true"),
					resource.TestCheckResourceAttrPair("data.spydersoftidentity_identity_resource.test", "id", "spydersoftidentity_identity_resource.test", "id"),
				),
			},
			{
				Config: testAccIdentityResourceConfig(name, `["name", "nickname"]`),
				Check:  resource.TestCheckResourceAttr("spydersoftidentity_identity_resource.test", "user_claims.#", "2"),
			},
			{
				ResourceName:      "spydersoftidentity_identity_resource.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}
