package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/provider"
)

// Acceptance tests run against a real identity server and create/destroy
// objects named tf-acc-*. They only run with TF_ACC=1 and the same
// SPYDERSOFT_IDENTITY_* variables the provider reads; see the README.

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"spydersoftidentity": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccProviderConfig leaves the provider block empty so it is configured
// from the environment.
const testAccProviderConfig = `provider "spydersoftidentity" {}
`

func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"SPYDERSOFT_IDENTITY_URL",
		"SPYDERSOFT_IDENTITY_TOKEN_URL",
		"SPYDERSOFT_IDENTITY_CLIENT_ID",
		"SPYDERSOFT_IDENTITY_CLIENT_SECRET",
	} {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for acceptance tests", v)
		}
	}
}

// testAccName returns a unique, recognisable object name for a test run.
func testAccName(kind string) string {
	return "tf-acc-" + kind + "-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
}
