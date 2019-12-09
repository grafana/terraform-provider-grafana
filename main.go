package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/terraform-providers/terraform-provider-grafana/grafana"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: grafana.Provider})
}
