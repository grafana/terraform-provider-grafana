package main

import (
	"os"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/provider"
)

func main() {
	p := provider.Provider("genimports") // Instantiate the provider so that all resources are registered
	_ = p

	if err := common.GenerateImportFiles(os.Args[1]); err != nil {
		panic(err)
	}
}
