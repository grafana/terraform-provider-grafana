package main

import (
	"context"
	"flag"
	"log"

	"github.com/grafana/terraform-provider-grafana/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-mux/tf6to5server"
)

//go:generate go run ./tools/genimports examples
//go:generate ./tools/generate-docs.sh

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

	// While we still have the SDK2 provider, we have to use the provider v5 protocol
	// See https://developer.hashicorp.com/terraform/plugin/mux/translating-protocol-version-6-to-5
	downgradedFrameworkProvider, err := tf6to5server.DowngradeServer(
		context.Background(),
		providerserver.NewProtocol6(provider.FrameworkProvider(version)),
	)
	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov5.ProviderServer{
		func() tfprotov5.ProviderServer {
			return downgradedFrameworkProvider
		},
		provider.Provider(version).GRPCProvider,
	}
	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt

	if debugMode {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	err = tf5server.Serve(
		"registry.terraform.io/grafana/grafana",
		muxServer.ProviderServer,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
