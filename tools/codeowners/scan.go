package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// scanGoFiles uses Go AST analysis to find resource registrations and map
// terraform resource names to their source files. It looks for calls to:
//   - common.NewLegacySDKResource(category, name, ...)
//   - common.NewResource(category, name, ...)
//   - common.NewLegacySDKDataSource(category, name, ...)
//   - common.NewDataSource(category, name, ...)
//
// The name argument is extracted either as a string literal or by resolving
// package-level const/var declarations.
//
// For appplatform resources whose names are computed dynamically via
// formatResourceType(kind), the name won't appear in a registration call.
// As a fallback, we extract the "kind" from the TF name and match against
// *_resource.go file names.
func scanGoFiles(root string, components []component) error {
	// Build a lookup from TF name to component indices
	tfNameToIndices := make(map[string][]int)
	for i, c := range components {
		tfNameToIndices[c.TFName] = append(tfNameToIndices[c.TFName], i)
	}

	// registrationFuncs are the selector names on "common" we look for
	registrationFuncs := map[string]bool{
		"NewLegacySDKResource":   true,
		"NewResource":            true,
		"NewLegacySDKDataSource": true,
		"NewDataSource":          true,
	}

	// Collect non-test .go files and track appplatform files for phase 2
	type goFileInfo struct {
		rel  string
		name string
	}
	var appplatformFiles []goFileInfo

	resourcesDir := filepath.Join(root, "internal", "resources")
	fset := token.NewFileSet()

	err := filepath.Walk(resourcesDir, func(dirPath string, dirInfo os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !dirInfo.IsDir() {
			return nil
		}

		// Parse the entire Go package in this directory (skips test files)
		pkgs, err := parser.ParseDir(fset, dirPath, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			return nil // skip unparseable directories
		}

		for _, pkg := range pkgs {
			// Phase 0: Collect all package-level string const/var declarations
			// so we can resolve identifier references in registration calls.
			stringConsts := make(map[string]string) // ident name -> string value
			for _, file := range pkg.Files {
				for _, decl := range file.Decls {
					gd, ok := decl.(*ast.GenDecl)
					if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
						continue
					}
					for _, spec := range gd.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok || len(vs.Names) != len(vs.Values) {
							continue
						}
						for i, val := range vs.Values {
							if lit, ok := val.(*ast.BasicLit); ok && lit.Kind == token.STRING {
								if s, err := strconv.Unquote(lit.Value); err == nil {
									stringConsts[vs.Names[i].Name] = s
								}
							}
						}
					}
				}
			}

			// Phase 1: Walk the AST looking for registration calls
			for filePath, file := range pkg.Files {
				rel, err := filepath.Rel(root, filePath)
				if err != nil {
					continue
				}

				// Track appplatform files for phase 2
				if strings.Contains(rel, "appplatform/") && strings.HasSuffix(filepath.Base(rel), "_resource.go") {
					appplatformFiles = append(appplatformFiles, goFileInfo{rel: rel, name: filepath.Base(rel)})
				}

				ast.Inspect(file, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					// Check if this is a common.New*() call
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					ident, ok := sel.X.(*ast.Ident)
					if !ok || ident.Name != "common" {
						return true
					}
					if !registrationFuncs[sel.Sel.Name] {
						return true
					}

					// The name is the 2nd argument (index 1)
					if len(call.Args) < 2 {
						return true
					}

					name := resolveStringArg(call.Args[1], stringConsts)
					if name == "" {
						return true
					}

					if indices, ok := tfNameToIndices[name]; ok {
						for _, idx := range indices {
							c := &components[idx]
							if strings.HasPrefix(rel, c.PkgDir+"/") {
								c.SourceFiles = append(c.SourceFiles, rel)
							}
						}
					}

					return true
				})
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Phase 2: Fallback for appplatform resources with dynamic names.
	// These use formatResourceType(kind) which computes the name at runtime.
	// Extract the "kind" from the TF name and match against file names.
	for i := range components {
		c := &components[i]
		if len(c.SourceFiles) > 0 || c.PkgName != "appplatform" {
			continue
		}

		kind := extractAppPlatformKind(c.TFName)
		if kind == "" {
			continue
		}

		kindNoUnderscore := strings.ReplaceAll(strings.ToLower(kind), "_", "")
		for _, gf := range appplatformFiles {
			if !strings.HasPrefix(gf.rel, c.PkgDir+"/") {
				continue
			}
			nameNoExt := strings.TrimSuffix(gf.name, "_resource.go")
			nameNormalized := strings.ReplaceAll(nameNoExt, "_", "")
			if strings.Contains(nameNormalized, kindNoUnderscore) {
				c.SourceFiles = append(c.SourceFiles, gf.rel)
			}
		}
	}

	return nil
}

// resolveStringArg extracts a string value from an AST expression.
// It handles string literals directly and resolves identifiers against
// known package-level const/var declarations.
func resolveStringArg(expr ast.Expr, consts map[string]string) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			s, err := strconv.Unquote(e.Value)
			if err == nil {
				return s
			}
		}
	case *ast.Ident:
		if val, ok := consts[e.Name]; ok {
			return val
		}
	}
	return ""
}

// extractAppPlatformKind extracts the "kind" portion from an appplatform TF name.
// Format: grafana_apps_{group}_{kind}_{version}
// Examples:
//
//	grafana_apps_alerting_alertrule_v0alpha1 → alertrule
//	grafana_apps_secret_keeper_v1beta1 → keeper
//	grafana_apps_secret_keeper_activation_v1beta1 → keeper_activation
//	grafana_apps_dashboard_dashboard_v1beta1 → dashboard
//	grafana_apps_generic_resource → "" (not a standard appplatform name)
func extractAppPlatformKind(tfName string) string {
	rest := strings.TrimPrefix(tfName, "grafana_apps_")
	if rest == tfName {
		return "" // not an appplatform resource
	}

	parts := strings.Split(rest, "_")
	if len(parts) < 3 {
		return ""
	}

	// Find the version part (last part starting with "v" + digit)
	versionIdx := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) > 1 && parts[i][0] == 'v' && parts[i][1] >= '0' && parts[i][1] <= '9' {
			versionIdx = i
			break
		}
	}

	if versionIdx <= 1 {
		return ""
	}

	// Kind is everything between the group (index 0) and the version
	return strings.Join(parts[1:versionIdx], "_")
}
