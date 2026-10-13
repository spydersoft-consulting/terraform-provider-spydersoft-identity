// Regenerate docs/ from the provider schema and examples/ (run `go generate ./...`).
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name spydersoftidentity --rendered-provider-name "Spydersoft Identity"

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/provider"
)

// version is set at release time via -ldflags.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/spydersoft-consulting/spydersoft-identity",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
