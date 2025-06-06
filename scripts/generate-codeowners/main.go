package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/grafanalabs/terraform-provider-grafana/scripts/generate-codeowners/pkg/generator"
)

const (
	generatedMarker = "# GENERATED BELOW (Regenerate with 'make CODEOWNERS')"
)

func main() {
	repoRoot, err := filepath.Abs("../../.github")
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Open the CODEOWNERS file for writing
	codeownersFilePath := filepath.Join(repoRoot, "CODEOWNERS")
	codeownersFile, err := os.OpenFile(codeownersFilePath, os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open CODEOWNERS file: %v", err)
	}
	defer func() {
		if err := codeownersFile.Close(); err != nil {
			log.Fatalf("Failed to close CODEOWNERS file: %v", err)
		}
	}()

	pathsToCheck := []string{
		"docs",
		filepath.Join("internal", "resources"),
	}

	generator := generator.New(repoRoot, codeownersFile)
	if err := generator.Generate(pathsToCheck); err != nil {
		log.Fatal(err)
	}
}
