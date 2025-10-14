package main

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
)

func main() {
	fmt.Println("Listing resources without lister functions:")
	for _, r := range provider.Resources() {
		if r.ListIDsFunc == nil {
			fmt.Println(r.Name)
		}
	}
}
