package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/provider"
)

// Acceptance tests drive the provider through a real Terraform/OpenTofu binary
// and only run with TF_ACC=1. By default they talk to an in-memory fake of the
// identity server (fakeapi_test.go), so they need no server and are safe to run
// anywhere. Set TF_ACC_LIVE=1 (plus the SPYDERSOFT_IDENTITY_* variables the
// provider reads) to run them against a real server instead; that creates and
// destroys objects named tf-acc-*. See the README.

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"spydersoftidentity": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccProviderConfig leaves the provider block empty so it is configured
// from the environment.
const testAccProviderConfig = `provider "spydersoftidentity" {}
`

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC_LIVE") != "1" {
		// Point the provider at a private fake, ignoring any real credentials
		// in the environment so a stray variable can never hit a live server.
		base := startFakeIdentity(t)
		t.Setenv("SPYDERSOFT_IDENTITY_URL", base)
		t.Setenv("SPYDERSOFT_IDENTITY_TOKEN_URL", base+"/connect/token")
		t.Setenv("SPYDERSOFT_IDENTITY_CLIENT_ID", "fake-client")
		t.Setenv("SPYDERSOFT_IDENTITY_CLIENT_SECRET", "fake-secret")
		return
	}

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
