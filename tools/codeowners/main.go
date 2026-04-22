// Command codeowners generates a CODEOWNERS file from Backstage catalog-info YAML files.
//
// It reads the root catalog-info.yaml, follows Location targets to per-package
// catalog-resource.yaml and catalog-data-source.yaml files, extracts ownership
// information, and generates .github/CODEOWNERS.
//
// Resource-to-file mapping uses Go AST analysis to find calls to
// common.NewLegacySDKResource, common.NewResource, common.NewLegacySDKDataSource,
// and common.NewDataSource, extracting the resource name argument (resolving
// string constants where needed) and recording the source file position.
//
// Usage:
//
//	go run ./tools/codeowners [flags]
//
// Flags:
//
//	--check     Compare generated output against existing .github/CODEOWNERS; exit 1 if different
//	--root DIR  Repository root directory (default: ".")
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// catalogEntity represents a Backstage catalog YAML document.
type catalogEntity struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Owner   string   `yaml:"owner"`
		Type    string   `yaml:"type"`
		Targets []string `yaml:"targets"`
	} `yaml:"spec"`
}

// component is a parsed catalog component with its resolved ownership and package location.
type component struct {
	Name    string // catalog name, e.g. "resource-grafana_dashboard"
	TFName  string // terraform name, e.g. "grafana_dashboard"
	Type    string // "terraform-resource" or "terraform-data-source"
	Owner   string // GitHub team name, e.g. "dashboards-squad"
	PkgDir  string // e.g. "internal/resources/grafana"
	PkgName string // e.g. "grafana"
	// Populated by scanGoFiles:
	SourceFiles []string // Go source files where this resource name appears (relative to root)
}

// rule is a single CODEOWNERS entry.
type rule struct {
	Pattern string
	Team    string
}

func main() {
	check := flag.Bool("check", false, "compare generated output against existing .github/CODEOWNERS; exit 1 if different")
	root := flag.String("root", ".", "repository root directory")
	flag.Parse()

	if err := run(*root, *check); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// readStaticRules reads the optional CODEOWNERS.in file. Returns empty string if
// the file doesn't exist.
func readStaticRules(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func run(root string, check bool) error {
	// Parse root catalog-info.yaml
	_, targets, err := parseRootCatalog(filepath.Join(root, "catalog-info.yaml"))
	if err != nil {
		return fmt.Errorf("parsing root catalog: %w", err)
	}

	// Parse all per-package catalog files
	var components []component
	for _, target := range targets {
		path := filepath.Join(root, filepath.Clean(target))
		comps, err := parseCatalogFile(path)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", target, err)
		}
		components = append(components, comps...)
	}

	// Scan Go source files to find which files contain each resource name
	if err := scanGoFiles(root, components); err != nil {
		return fmt.Errorf("scanning Go files: %w", err)
	}

	// Validate
	var warnings []string
	for _, c := range components {
		if c.Owner == "" {
			warnings = append(warnings, fmt.Sprintf("WARNING: empty owner for %s in %s", c.Name, c.PkgDir))
		}
		if len(c.SourceFiles) == 0 {
			warnings = append(warnings, fmt.Sprintf("WARNING: no Go source files found for %s (%s)", c.Name, c.TFName))
		}
	}

	// Read static rules from CODEOWNERS.in (optional)
	staticContent, err := readStaticRules(filepath.Join(root, ".github", "CODEOWNERS.in"))
	if err != nil {
		return fmt.Errorf("reading CODEOWNERS.in: %w", err)
	}

	// Generate rules
	rules := generateRules(root, components)

	// Format output
	output := formatOutput(staticContent, rules)

	// Print warnings to stderr
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	// Report files falling through to default rule
	reportDefaultCoverage(os.Stderr, root, staticContent, rules)

	if check {
		existing, err := os.ReadFile(filepath.Join(root, ".github", "CODEOWNERS"))
		if err != nil {
			return fmt.Errorf("reading existing CODEOWNERS: %w", err)
		}
		if !bytes.Equal(existing, []byte(output)) {
			fmt.Fprintln(os.Stderr, "CODEOWNERS is out of date. Regenerate with: make codeowners")
			fmt.Fprintln(os.Stderr, "--- expected (generated)")
			fmt.Fprintln(os.Stderr, "+++ actual (.github/CODEOWNERS)")
			diffLines(os.Stderr, output, string(existing))
			return fmt.Errorf("CODEOWNERS is out of date")
		}
		fmt.Fprintln(os.Stderr, "CODEOWNERS is up to date.")
		return nil
	}

	fmt.Print(output)
	return nil
}
