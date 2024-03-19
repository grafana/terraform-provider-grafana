package main

import (
	"os"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/pkg/provider"
)

// TODO: Move to cmd, and remove global var in common

func main() {
	p := provider.Provider("genimports") // Instantiate the provider so that all resources are registered
	_ = p

	if err := common.GenerateImportFiles(os.Args[1]); err != nil {
		panic(err)
	}
}
