package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/grafana/terraform-provider-grafana/scripts/add-resource/codegen"
)

func main() {
	// Step 1: Collect the manifest URL.
	var manifestURL string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Manifest URL").
				Description("Raw GitHub URL for a Grafana manifest.cue file.\nExample: https://raw.githubusercontent.com/grafana/grafana/refs/heads/main/apps/advisor/kinds/manifest.cue").
				Validate(func(s string) error {
					u, err := url.ParseRequestURI(s)
					if err != nil {
						return err
					}
					if !strings.HasSuffix(path.Base(u.Path), ".cue") {
						return fmt.Errorf("must be a .cue file")
					}
					return nil
				}).
				Value(&manifestURL),
		),
	).Run(); err != nil {
		panic(fmt.Errorf("form error: %w", err))
	}

	// Step 2: Fetch and parse the manifest.
	fmt.Println("Fetching manifest…")
	manifest, err := codegen.FetchAndParseManifest(manifestURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse manifest: %w", err))
	}
	fmt.Printf("App: %s  Group: %s\n", manifest.AppName, manifest.GroupOverride)

	// Step 3: Select version.
	versions := manifest.VersionNames()
	if len(versions) == 0 {
		panic("no versions found in manifest")
	}
	var selectedVersion string
	if len(versions) == 1 {
		selectedVersion = versions[0]
		fmt.Printf("Version: %s (only one available)\n", selectedVersion)
	} else {
		opts := make([]huh.Option[string], len(versions))
		for i, v := range versions {
			opts[i] = huh.NewOption(v, v)
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select version").
					Description("Available versions from the manifest.").
					Options(opts...).
					Value(&selectedVersion),
			),
		).Run(); err != nil {
			panic(fmt.Errorf("form error: %w", err))
		}
	}

	// Step 4: Select kind.
	kinds := manifest.Versions[selectedVersion]
	if len(kinds) == 0 {
		panic(fmt.Sprintf("no kinds found for version %s", selectedVersion))
	}
	var selectedKind string
	if len(kinds) == 1 {
		selectedKind = kinds[0]
		fmt.Printf("Kind: %s (only one available)\n", selectedKind)
	} else {
		opts := make([]huh.Option[string], len(kinds))
		for i, k := range kinds {
			opts[i] = huh.NewOption(k, k)
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select kind").
					Description(fmt.Sprintf("Available kinds for version %s.", selectedVersion)).
					Options(opts...).
					Value(&selectedKind),
			),
		).Run(); err != nil {
			panic(fmt.Errorf("form error: %w", err))
		}
	}

	// Step 5: Find the kind file and extract kind metadata.
	fmt.Printf("Looking up kind %q…\n", selectedKind)
	kindInfo, err := codegen.FetchKindInfo(selectedKind, manifest)
	if err != nil {
		panic(fmt.Errorf("failed to find kind %q: %w", selectedKind, err))
	}
	fmt.Printf("Kind: %s  Plural: %s  File: %s\n", kindInfo.KindName, kindInfo.PluralName, kindInfo.FileURL)

	// Step 6: Ask for schema name, output directory, and formatting option.
	var schemaName, outputDir string
	var skipFormatting bool

	kindNameDefault := kindInfo.KindName
	if kindNameDefault == "" {
		kindNameDefault = selectedKind
	}

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Schema name").
				Description("Name used as the forced_envelope and resource constructor. Example: Check").
				Placeholder(kindNameDefault).
				Validate(notEmpty()).
				Value(&schemaName),

			huh.NewInput().
				Title("Output directory").
				Description("Directory under internal/resources/ where the file is written. Example: appplatform").
				Placeholder("appplatform").
				Validate(notEmpty()).
				Value(&outputDir),

			huh.NewConfirm().
				Title("Skip formatting").
				Description("Disable gofmt after generation. Useful when debugging template output.").
				Value(&skipFormatting),
		),
	).Run(); err != nil {
		panic(fmt.Errorf("form error: %w", err))
	}

	// Step 7: Warn if the resource scaffold already exists.
	name := strings.ToLower(schemaName)
	scaffoldFile := filepath.Join("..", "..", "internal", "resources", outputDir, name+"_resource.go")
	if _, err := os.Stat(scaffoldFile); err == nil {
		fmt.Printf("\nWARNING: %s already exists and will be overwritten.\n", scaffoldFile)
		var proceed bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Overwrite existing file?").
					Description("The resource scaffold file already exists. Overwriting it will discard any manual edits.").
					Value(&proceed),
			),
		).Run(); err != nil {
			panic(fmt.Errorf("form error: %w", err))
		}
		if !proceed {
			fmt.Println("Aborted.")
			return
		}
	}

	// Step 8: Generate.
	fmt.Println("Generating…")
	if err := codegen.Generate(&codegen.Config{
		PackageValue:   &manifest.PackageValue,
		Subpath:        kindInfo.SpecSubpath,
		Name:           schemaName,
		KindName:       kindInfo.KindName,
		PluralName:     kindInfo.PluralName,
		Version:        selectedVersion,
		AppName:        manifest.AppName,
		ServiceName:    manifest.ServiceName,
		GroupOverride:  manifest.GroupOverride,
		OutputDir:      outputDir,
		SkipFormatting: skipFormatting,
	}); err != nil {
		panic(fmt.Errorf("generation failed: %w", err))
	}

	fmt.Printf("\nDone!\n")
	fmt.Printf("  internal/resources/%s/%s_types_gen.go  (generated types — do not edit)\n", outputDir, name)
	fmt.Printf("  internal/resources/%s/%s_resource.go   (scaffold — edit if you need more customization)\n", outputDir, name)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Modify %s_resource.go if you need more customization.\n", name)
	fmt.Printf("  2. Register the resource in internal/resources/appplatform/catalog-resource.yaml.\n")
}

func notEmpty() func(s string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("required")
		}
		return nil
	}
}
