package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// packageGroup holds all components for a single package directory.
type packageGroup struct {
	pkgDir  string
	pkgName string
	comps   []component
}

// generateRules produces CODEOWNERS rules from the parsed components.
// root is the repository root, used to check whether example/doc paths exist on disk.
func generateRules(root string, components []component) []rule {
	// Group components by package
	pkgMap := make(map[string]*packageGroup)
	var pkgOrder []string
	for _, c := range components {
		pg, ok := pkgMap[c.PkgDir]
		if !ok {
			pg = &packageGroup{pkgDir: c.PkgDir, pkgName: c.PkgName}
			pkgMap[c.PkgDir] = pg
			pkgOrder = append(pkgOrder, c.PkgDir)
		}
		pg.comps = append(pg.comps, c)
	}
	sort.Strings(pkgOrder)

	var rules []rule
	for _, pkgDir := range pkgOrder {
		pg := pkgMap[pkgDir]
		pkgRules := generatePackageRules(pg)
		rules = append(rules, pkgRules...)
	}

	// Generate rules for examples/ and docs/ directories
	rules = append(rules, generateExampleAndDocRules(root, components)...)

	return rules
}

// generatePackageRules generates CODEOWNERS rules for a single package.
func generatePackageRules(pg *packageGroup) []rule {
	// Count owners
	ownerCount := make(map[string]int)
	for _, c := range pg.comps {
		if c.Owner != "" {
			ownerCount[c.Owner]++
		}
	}

	// If single owner, just emit a wildcard
	if len(ownerCount) == 1 {
		var owner string
		for o := range ownerCount {
			owner = o
		}
		return []rule{{
			Pattern: fmt.Sprintf("/%s/**", pg.pkgDir),
			Team:    fmt.Sprintf("@grafana/%s", owner),
		}}
	}

	// Multi-owner: find the majority owner for the wildcard fallback
	majorityOwner := findMajorityOwner(ownerCount)

	// Group components by owner (excluding majority)
	ownerComponents := make(map[string][]component)
	for _, c := range pg.comps {
		if c.Owner != "" && c.Owner != majorityOwner {
			ownerComponents[c.Owner] = append(ownerComponents[c.Owner], c)
		}
	}

	var rules []rule

	// CODEOWNERS is last-match-wins: wildcard first, specific overrides after
	rules = append(rules, rule{
		Pattern: fmt.Sprintf("/%s/**", pg.pkgDir),
		Team:    fmt.Sprintf("@grafana/%s", majorityOwner),
	})

	// Emit specific patterns for minority owners, derived from actual source files
	var specificRules []rule
	for owner, comps := range ownerComponents {
		patterns := sourceFilePatterns(comps)
		for _, pattern := range patterns {
			specificRules = append(specificRules, rule{
				Pattern: pattern,
				Team:    fmt.Sprintf("@grafana/%s", owner),
			})
		}
	}

	// Sort specific rules by pattern for deterministic output
	sort.Slice(specificRules, func(i, j int) bool {
		return specificRules[i].Pattern < specificRules[j].Pattern
	})

	rules = append(rules, specificRules...)
	return rules
}

// sourceFilePatterns generates CODEOWNERS glob patterns from the actual source
// files found for a set of components.
func sourceFilePatterns(comps []component) []string {
	seen := make(map[string]bool)
	var patterns []string

	for _, c := range comps {
		for _, f := range c.SourceFiles {
			pattern := fileBasePattern(f)
			if !seen[pattern] {
				seen[pattern] = true
				patterns = append(patterns, pattern)
			}
		}
	}

	sort.Strings(patterns)
	return patterns
}

// findMajorityOwner returns the owner with the most components.
func findMajorityOwner(ownerCount map[string]int) string {
	var maxOwner string
	var maxCount int
	owners := make([]string, 0, len(ownerCount))
	for o := range ownerCount {
		owners = append(owners, o)
	}
	sort.Strings(owners)

	for _, o := range owners {
		if ownerCount[o] > maxCount {
			maxOwner = o
			maxCount = ownerCount[o]
		}
	}
	return maxOwner
}

// fileBasePattern extracts a CODEOWNERS glob pattern from a Go source file path.
// Given "internal/resources/grafana/resource_alerting_contact_point.go",
// returns "/internal/resources/grafana/resource_alerting_contact_point*"
// which matches the .go file, _test.go, and related files.
func fileBasePattern(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	// Strip .go extension
	base = strings.TrimSuffix(base, ".go")
	return fmt.Sprintf("/%s/%s*", dir, base)
}

// generateExampleAndDocRules produces CODEOWNERS rules for:
//   - examples/resources/<tf_name>/  (directories)
//   - examples/data-sources/<tf_name>/  (directories)
//   - docs/resources/<tf_name_without_grafana_prefix>.md  (files)
//   - docs/data-sources/<tf_name_without_grafana_prefix>.md  (files)
//
// Only paths that exist on disk get rules. The root parameter is the
// repository root used to stat these paths.
func generateExampleAndDocRules(root string, components []component) []rule {
	var exampleRules, docRules []rule

	for _, c := range components {
		if c.Owner == "" {
			continue
		}
		team := fmt.Sprintf("@grafana/%s", c.Owner)

		// Determine example and doc paths based on component type
		var exampleDir, docFile string
		switch c.Type {
		case "terraform-resource":
			exampleDir = filepath.Join("examples", "resources", c.TFName)
			docFile = filepath.Join("docs", "resources", strings.TrimPrefix(c.TFName, "grafana_")+".md")
		case "terraform-data-source":
			exampleDir = filepath.Join("examples", "data-sources", c.TFName)
			docFile = filepath.Join("docs", "data-sources", strings.TrimPrefix(c.TFName, "grafana_")+".md")
		default:
			continue
		}

		// Example directory: only emit if it exists on disk
		if fi, err := os.Stat(filepath.Join(root, exampleDir)); err == nil && fi.IsDir() {
			exampleRules = append(exampleRules, rule{
				Pattern: fmt.Sprintf("/%s/*", exampleDir),
				Team:    team,
			})
		}

		// Doc file: only emit if it exists on disk
		if fi, err := os.Stat(filepath.Join(root, docFile)); err == nil && !fi.IsDir() {
			docRules = append(docRules, rule{
				Pattern: fmt.Sprintf("/%s", docFile),
				Team:    team,
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(exampleRules, func(i, j int) bool {
		return exampleRules[i].Pattern < exampleRules[j].Pattern
	})
	sort.Slice(docRules, func(i, j int) bool {
		return docRules[i].Pattern < docRules[j].Pattern
	})

	// Examples first, then docs
	var rules []rule
	rules = append(rules, exampleRules...)
	rules = append(rules, docRules...)
	return rules
}
