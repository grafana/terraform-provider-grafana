package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseRootCatalog reads the root catalog-info.yaml and returns the default owner
// and all Location target paths.
func parseRootCatalog(path string) (string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	var defaultOwner string
	var targets []string

	for {
		var entity catalogEntity
		err := dec.Decode(&entity)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("decoding YAML: %w", err)
		}

		switch entity.Kind {
		case "Component":
			defaultOwner = extractTeamName(entity.Spec.Owner)
		case "Location":
			targets = append(targets, entity.Spec.Targets...)
		}
	}

	return defaultOwner, targets, nil
}

// parseCatalogFile reads a catalog-resource.yaml or catalog-data-source.yaml
// and returns the components defined in it.
func parseCatalogFile(path string) ([]component, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Derive package info from path
	dir := filepath.Dir(path)
	pkgDir, pkgName := extractPkgInfo(dir)

	dec := yaml.NewDecoder(f)
	var components []component

	for {
		var entity catalogEntity
		err := dec.Decode(&entity)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}

		if entity.Kind != "Component" {
			continue
		}

		// Extract terraform name from catalog component name
		tfName := entity.Metadata.Name
		tfName = strings.TrimPrefix(tfName, "resource-")
		tfName = strings.TrimPrefix(tfName, "datasource-")

		components = append(components, component{
			Name:    entity.Metadata.Name,
			TFName:  tfName,
			Type:    entity.Spec.Type,
			Owner:   extractTeamName(entity.Spec.Owner),
			PkgDir:  pkgDir,
			PkgName: pkgName,
		})
	}

	return components, nil
}

// extractTeamName strips "group:default/" prefix from a Backstage owner string.
func extractTeamName(owner string) string {
	return strings.TrimPrefix(owner, "group:default/")
}

// extractPkgInfo extracts the relative package directory and short package name.
func extractPkgInfo(dir string) (string, string) {
	const marker = "internal/resources/"
	idx := strings.Index(dir, marker)
	if idx == -1 {
		return dir, filepath.Base(dir)
	}
	relDir := dir[idx:]
	pkgName := strings.TrimPrefix(relDir, marker)
	if i := strings.Index(pkgName, "/"); i != -1 {
		pkgName = pkgName[:i]
	}
	return relDir, pkgName
}
