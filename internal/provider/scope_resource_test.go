package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccScopeConfig(name, displayName, claims, props string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "spydersoftidentity_scope" "test" {
  name         = %[1]q
  display_name = %[2]q
  user_claims  = %[3]s
  properties   = %[4]s
}

data "spydersoftidentity_scope" "test" {
  name = spydersoftidentity_scope.test.name
}
`, name, displayName, claims, props)
}

func TestAccScope(t *testing.T) {
	name := testAccName("scope")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScopeConfig(name, "First", `["email", "name"]`, `{ owner = "acc" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "name", name),
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "enabled", "true"),
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "user_claims.#", "2"),
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "properties.owner", "acc"),
					resource.TestCheckResourceAttrPair("data.spydersoftidentity_scope.test", "id", "spydersoftidentity_scope.test", "id"),
					resource.TestCheckResourceAttr("data.spydersoftidentity_scope.test", "display_name", "First"),
					resource.TestCheckResourceAttr("data.spydersoftidentity_scope.test", "user_claims.#", "2"),
				),
			},
			{
				// Update in place: change a claim and a property.
				Config: testAccScopeConfig(name, "Second", `["email", "profile"]`, `{ owner = "acc2", team = "x" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "display_name", "Second"),
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "user_claims.#", "2"),
					resource.TestCheckTypeSetElemAttr("spydersoftidentity_scope.test", "user_claims.*", "profile"),
					resource.TestCheckResourceAttr("spydersoftidentity_scope.test", "properties.%", "2"),
				),
			},
			{
				ResourceName:                         "spydersoftidentity_scope.test",
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}
