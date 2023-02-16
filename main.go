package main

import (
	"context"
	"flag"
	"log"

	"github.com/grafana/terraform-provider-grafana/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

//go:generate ./tools/generate-docs.sh

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: provider.Provider(version), Debug: debugMode}

	if debugMode {
		err := plugin.Debug(context.Background(), "grafana.com/grafana/grafana", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	opts.ProviderAddr = "registry.terraform.io/grafana/grafana"
	plugin.Serve(opts)
}
