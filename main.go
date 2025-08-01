package main

import (
	"context"
	"flag"
	"log"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
)

//go:generate go run ./tools/genimports examples
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
//go:generate go run ./tools/setcategories docs

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"
)

// Mux config taken here: https://developer.hashicorp.com/terraform/plugin/framework/migrating/mux#terraform-0-12-compatibility-example
// While not every resource is migrated to the Terraform Plugin Framework, we must mux the old and new providers together.
func main() {
	ctx := context.Background()

	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	muxServer, err := provider.MakeProviderServer(ctx, version)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt

	if debugMode {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	err = tf5server.Serve(
		"registry.terraform.io/grafana/grafana",
		func() tfprotov5.ProviderServer { return muxServer },
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
