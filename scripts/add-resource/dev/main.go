package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/terraform-provider-grafana/scripts/add-resource/codegen"
)

func main() {
	manifestURL := "https://raw.githubusercontent.com/grafana/grafana/refs/heads/main/apps/alerting/rules/kinds/manifest.cue"
	version := "v0alpha1"
	selectedKind := "AlertRule"
	goTypesImportPath := "github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"

	fmt.Println("Fetching manifest…")
	manifest, err := codegen.FetchAndParseManifest(manifestURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse manifest: %w", err))
	}

	fmt.Println()
	codegen.PrintManifestSummary(manifest)
	fmt.Println()

	kindInfo, err := codegen.FindKind(manifest, selectedKind, version)
	if err != nil {
		panic(fmt.Errorf("failed to find kind %q: %w", selectedKind, err))
	}

	fmt.Println()
	codegen.PrintKindSummary(kindInfo)
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	manifestProperties := manifest.Properties()
	config := codegen.Config{
		PackageValue:      &kindInfo.Schema,
		KindName:          kindInfo.Kind,
		PluralName:        kindInfo.PluralName,
		Version:           version,
		AppName:           manifestProperties.AppName,
		GroupOverride:     manifestProperties.FullGroup,
		GoTypesImportPath: goTypesImportPath,
		ResourceOutputDir: "appplatform",
		SkipFormatting:    true,
		GrafanaVersion:    ">=11.0.0",
		IsEnterprise:      false,
		TemplatesDir:      filepath.Join(cwd, "..", "codegen", "templates"),
		OutputDir:         filepath.Join(cwd, "..", "..", ".."),
	}

	fmt.Println("Generating…")
	if err := codegen.Generate(context.Background(), config); err != nil {
		panic(fmt.Errorf("generation failed: %w", err))
	}
}
