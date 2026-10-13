package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccRoleConfig(name, claims string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "spydersoftidentity_role" "test" {
  name   = %[1]q
  claims = %[2]s
}

data "spydersoftidentity_role" "test" {
  name = spydersoftidentity_role.test.name
}
`, name, claims)
}

func TestAccRole(t *testing.T) {
	name := testAccName("role")
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Two claims share a type; one type contains colons.
				Config: testAccRoleConfig(name, `[
				  { type = "perm", value = "read" },
				  { type = "perm", value = "write" },
				  { type = "urn:example:claims:dept", value = "eng" },
				]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_role.test", "name", name),
					resource.TestCheckResourceAttrSet("spydersoftidentity_role.test", "id"),
					resource.TestCheckResourceAttr("spydersoftidentity_role.test", "claims.#", "3"),
					resource.TestCheckResourceAttrPair("data.spydersoftidentity_role.test", "id", "spydersoftidentity_role.test", "id"),
					resource.TestCheckResourceAttr("data.spydersoftidentity_role.test", "claims.#", "3"),
				),
			},
			{
				// Drop one of the two "perm" claims (the API can only delete by
				// type), drop the urn-style one, add another, and rename the role.
				Config: testAccRoleConfig(renamed, `[
				  { type = "perm", value = "write" },
				  { type = "team", value = "platform" },
				]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_role.test", "name", renamed),
					resource.TestCheckResourceAttr("spydersoftidentity_role.test", "claims.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("spydersoftidentity_role.test", "claims.*", map[string]string{"type": "perm", "value": "write"}),
					resource.TestCheckTypeSetElemNestedAttrs("spydersoftidentity_role.test", "claims.*", map[string]string{"type": "team", "value": "platform"}),
				),
			},
			{
				// A claim with an empty value is allowed.
				Config: testAccRoleConfig(renamed, `[{ type = "flag" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("spydersoftidentity_role.test", "claims.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("spydersoftidentity_role.test", "claims.*", map[string]string{"type": "flag", "value": ""}),
				),
			},
			{
				ResourceName:      "spydersoftidentity_role.test",
				ImportState:       true,
				ImportStateId:     renamed,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRole_RejectsClaimTypeWithSlash(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig(testAccName("role"), `[{ type = "http://schemas.example.com/claims/dept", value = "eng" }]`),
				ExpectError: regexp.MustCompile(`(?s)Unsupported claim type.*cannot remove such a claim`),
			},
		},
	})
}
