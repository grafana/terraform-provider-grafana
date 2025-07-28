package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
)

func main() {
	examplesPath := os.Args[1]
	if err := generateImportFiles(examplesPath); err != nil {
		panic(err)
	}
}

// GenerateImportFiles generates import files for all resources that use a helper defined in this package
func generateImportFiles(examplesPath string) error {
	for _, r := range provider.Resources() {
		resourcePath := filepath.Join(examplesPath, "resources", r.Name, "import.sh")
		if err := os.RemoveAll(resourcePath); err != nil { // Remove the file if it exists
			return err
		}

		if r.IDType == nil {
			log.Printf("Skipping import file generation for %s because it does not have an ID type\n", r.Name)
			continue
		}

		log.Printf("Generating import file for %s (writing to %s)\n", r.Name, resourcePath)
		if err := os.WriteFile(resourcePath, []byte(r.ImportExample()), 0600); err != nil {
			return err
		}
	}
	return nil
}
