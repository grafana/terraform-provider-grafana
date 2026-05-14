package main

import (
	"fmt"
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

	var appplatformFiles []appplatformFile

	resourcesDir := filepath.Join(root, "internal", "resources")
	fset := token.NewFileSet()

	err := filepath.Walk(resourcesDir, func(dirPath string, dirInfo os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: scanning %s: %v\n", dirPath, err)
			return nil
		}
		if !dirInfo.IsDir() {
			return nil
		}

		pkgs, err := parser.ParseDir(fset, dirPath, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: parsing %s: %v\n", dirPath, err)
			return nil
		}

		for _, pkg := range pkgs {
			stringConsts := collectStringConsts(pkg.Files)
			scanPackageFiles(root, pkg.Files, stringConsts, registrationFuncs, tfNameToIndices, components, &appplatformFiles)
		}

		return nil
	})
	if err != nil {
		return err
	}

	resolveAppPlatformFallbacks(components, appplatformFiles)
	return nil
}

// appplatformFile tracks a *_resource.go file in the appplatform package.
type appplatformFile struct {
	rel  string
	name string
}

// collectStringConsts collects all package-level string const/var declarations
// from parsed Go files so we can resolve identifier references.
func collectStringConsts(files map[string]*ast.File) map[string]string {
	consts := make(map[string]string)
	for _, file := range files {
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
							consts[vs.Names[i].Name] = s
						}
					}
				}
			}
		}
	}
	return consts
}

// scanPackageFiles walks AST files looking for registration calls
// and records source file mappings for matched components.
func scanPackageFiles(
	root string,
	files map[string]*ast.File,
	stringConsts map[string]string,
	registrationFuncs map[string]bool,
	tfNameToIndices map[string][]int,
	components []component,
	appplatformFiles *[]appplatformFile,
) {
	for filePath, file := range files {
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			continue
		}

		if strings.Contains(rel, "appplatform/") && strings.HasSuffix(filepath.Base(rel), "_resource.go") {
			*appplatformFiles = append(*appplatformFiles, appplatformFile{rel: rel, name: filepath.Base(rel)})
		}

		ast.Inspect(file, func(n ast.Node) bool {
			name := extractRegistrationName(n, registrationFuncs, stringConsts)
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

// extractRegistrationName checks if an AST node is a common.New*() registration
// call and returns the resource name argument, or "" if not a match.
func extractRegistrationName(n ast.Node, registrationFuncs map[string]bool, stringConsts map[string]string) string {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "common" {
		return ""
	}
	if !registrationFuncs[sel.Sel.Name] {
		return ""
	}
	if len(call.Args) < 2 {
		return ""
	}
	return resolveStringArg(call.Args[1], stringConsts)
}

// resolveAppPlatformFallbacks handles appplatform resources whose names are
// computed dynamically via formatResourceType(kind). It extracts the "kind"
// from the TF name and matches against *_resource.go file names.
func resolveAppPlatformFallbacks(components []component, appplatformFiles []appplatformFile) {
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
