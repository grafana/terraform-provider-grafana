package codegen

import (
	"context"
	"fmt"
	"slices"
	"strings"

	cogcue "github.com/grafana/cog/pkg/cue"
	"github.com/grafana/grafana-app-sdk/codegen"
	"github.com/grafana/grafana-app-sdk/codegen/cuekind"
)

// FetchAndParseManifest compiles the kinds package at rawURL and extracts
// app metadata and versions/kinds using the CUE value API.
func FetchAndParseManifest(rawURL string) (codegen.AppManifest, error) {
	parsed, err := cogcue.Parse(context.Background(), cogcue.Input{
		URL: rawURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compile kinds package: %w", err)
	}

	cueKind, err := cuekind.FromCueValue(parsed.Value)
	if err != nil {
		return nil, fmt.Errorf("could not create App manifest CUE kind: %w", err)
	}

	parser, err := cuekind.NewParser(cueKind, false)
	if err != nil {
		return nil, fmt.Errorf("could not create App manifest parser: %w", err)
	}

	appManifest, err := parser.ParseManifest(cuekind.DefaultManifestSelector)
	if err != nil {
		return nil, fmt.Errorf("could not parse App manifest: %w", err)
	}

	return appManifest, nil
}

func ManifestVersions(manifest codegen.AppManifest) []string {
	versionNames := make([]string, 0, len(manifest.Versions()))
	for _, version := range manifest.Versions() {
		versionNames = append(versionNames, version.Name())
	}

	slices.Sort(versionNames)

	return versionNames
}

func PrintManifestSummary(manifest codegen.AppManifest) {
	properties := manifest.Properties()
	versions := strings.Join(ManifestVersions(manifest), ", ")

	fmt.Printf("⦁ App:      %s\n⦁ Group:    %s\n⦁ Versions: %s\n", properties.AppName, properties.FullGroup, versions)
}

func KindsForVersion(manifest codegen.AppManifest, version string) []codegen.VersionedKind {
	for _, v := range manifest.Versions() {
		if v.Name() != version {
			continue
		}

		return v.Kinds()
	}

	return nil
}

func FindKind(manifest codegen.AppManifest, kindName string, version string) (codegen.VersionedKind, error) {
	kinds := KindsForVersion(manifest, version)

	for _, kind := range kinds {
		if kind.Kind != kindName {
			continue
		}

		return kind, nil
	}

	return codegen.VersionedKind{}, fmt.Errorf("kind %q not found with version %q", kindName, version)
}

func PrintKindSummary(kind codegen.VersionedKind) {
	fmt.Printf("⦁ Kind:   %s\n⦁ Plural: %s\n", kind.Kind, kind.PluralName)
}
