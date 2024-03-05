package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func main() {
	path := os.Getenv("TFGEN_OUT_PATH")
	if path == "" {
		log.Fatal("TFGEN_OUT_PATH environment variable must be set")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	items, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, item := range items {
		if item.IsDir() {
			continue
		}

		if !strings.HasSuffix(item.Name(), ".tf") {
			continue
		}

		fpath := filepath.Join(path, item.Name())

		err := common.StripDefaults(fpath, nil)
		if err != nil {
			panic(err)
		}
	}
}
