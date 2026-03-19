package codegen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
)

// ManifestInfo holds extracted metadata from a Grafana manifest.cue.
type ManifestInfo struct {
	AppName       string
	GroupOverride string
	// Versions maps version names (e.g. "v0alpha1") to kind identifier slices.
	Versions map[string][]string
	// BaseURL is the directory URL for sibling kind files.
	BaseURL string
	// RawURL is the original manifest URL.
	RawURL string
}

// VersionNames returns sorted version names from the manifest.
func (m *ManifestInfo) VersionNames() []string {
	names := make([]string, 0, len(m.Versions))
	for k := range m.Versions {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// KindInfo holds extracted metadata for a single kind from a CUE file.
type KindInfo struct {
	// Identifier is the CUE identifier in the manifest (e.g. "checkv0alpha1").
	Identifier string
	// KindName is the string value of the "kind" field (e.g. "Check").
	KindName string
	// PluralName is the string value of the "pluralName" field (e.g. "checks").
	PluralName string
	// SpecSubpath is the CUE path to the spec value (e.g. "checkv0alpha1.schema.spec").
	SpecSubpath string
	// FileURL is the URL of the CUE file that defines this kind.
	FileURL string
}

// FetchAndParseManifest fetches a manifest.cue from a raw GitHub URL and
// extracts versions and kind identifiers.
func FetchAndParseManifest(rawURL string) (*ManifestInfo, error) {
	data, err := FetchURL(rawURL)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseFile("manifest.cue", data, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	info := &ManifestInfo{
		Versions: make(map[string][]string),
		RawURL:   rawURL,
		BaseURL:  dirURL(rawURL),
	}

	manifestExpr := findFieldExpr(file.Decls, "manifest")
	if manifestExpr == nil {
		return nil, fmt.Errorf("manifest field not found in CUE file")
	}
	manifestStruct, ok := manifestExpr.(*ast.StructLit)
	if !ok {
		return nil, fmt.Errorf("manifest field is not a struct")
	}

	info.AppName = findStringField(manifestStruct.Elts, "appName")
	info.GroupOverride = findStringField(manifestStruct.Elts, "groupOverride")

	versionsExpr := findFieldExpr(manifestStruct.Elts, "versions")
	if versionsExpr == nil {
		return nil, fmt.Errorf("versions field not found in manifest")
	}
	versionsStruct, ok := versionsExpr.(*ast.StructLit)
	if !ok {
		return nil, fmt.Errorf("versions is not a struct")
	}

	for _, d := range versionsStruct.Elts {
		f, ok := d.(*ast.Field)
		if !ok {
			continue
		}
		versionName := labelStr(f.Label)
		if versionName == "" {
			continue
		}
		vStruct, ok := f.Value.(*ast.StructLit)
		if !ok {
			continue
		}
		kindsExpr := findFieldExpr(vStruct.Elts, "kinds")
		if kindsExpr == nil {
			continue
		}
		kindsList, ok := kindsExpr.(*ast.ListLit)
		if !ok {
			continue
		}
		var kinds []string
		for _, e := range kindsList.Elts {
			if ident, ok := e.(*ast.Ident); ok {
				kinds = append(kinds, ident.Name)
			}
		}
		info.Versions[versionName] = kinds
	}

	return info, nil
}

// versionSuffixRe matches the version suffix at the end of a CUE kind identifier
// (e.g., "v0alpha1" in "checkv0alpha1").
var versionSuffixRe = regexp.MustCompile(`v\d+(alpha|beta)\d*$`)

// FetchKindInfo finds and parses the CUE file that defines the given kind
// identifier. It tries several candidate URLs in order.
func FetchKindInfo(kindIdentifier, baseURL, version string) (*KindInfo, error) {
	stripped := versionSuffixRe.ReplaceAllString(kindIdentifier, "")
	candidates := []string{
		baseURL + stripped + ".cue",
		baseURL + kindIdentifier + ".cue",
		baseURL + version + "/" + stripped + ".cue",
		baseURL + version + "/" + kindIdentifier + ".cue",
	}

	for _, url := range candidates {
		data, err := FetchURL(url)
		if err != nil {
			continue
		}
		info, err := parseKindFromData(kindIdentifier, url, data)
		if err != nil {
			continue
		}
		return info, nil
	}
	return nil, fmt.Errorf("could not find CUE file for kind %q; tried: %v", kindIdentifier, candidates)
}

func parseKindFromData(kindIdentifier, fileURL string, data []byte) (*KindInfo, error) {
	file, err := parser.ParseFile("kind.cue", data, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CUE: %w", err)
	}

	kindExpr := findFieldExpr(file.Decls, kindIdentifier)
	if kindExpr == nil {
		return nil, fmt.Errorf("identifier %q not found in file", kindIdentifier)
	}
	kindStruct, ok := kindExpr.(*ast.StructLit)
	if !ok {
		return nil, fmt.Errorf("kind %q is not a struct", kindIdentifier)
	}

	return &KindInfo{
		Identifier:  kindIdentifier,
		KindName:    findStringField(kindStruct.Elts, "kind"),
		PluralName:  findStringField(kindStruct.Elts, "pluralName"),
		SpecSubpath: kindIdentifier + ".schema.spec",
		FileURL:     fileURL,
	}, nil
}

// LoadCueValue fetches a CUE file from a URL and compiles it to a cue.Value.
func LoadCueValue(url string) (cue.Value, error) {
	data, err := FetchURL(url)
	if err != nil {
		return cue.Value{}, err
	}
	v := cuecontext.New().CompileString(string(data))
	if v.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to compile CUE: %w", v.Err())
	}
	return v, nil
}

// FetchURL performs an HTTP GET request, using GITHUB_PAT env var if set.
func FetchURL(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if pat := os.Getenv("GITHUB_PAT"); pat != "" {
		req.Header.Set("Authorization", "Bearer "+pat)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// dirURL returns the directory portion of a URL (up to and including the last slash).
func dirURL(rawURL string) string {
	i := strings.LastIndex(rawURL, "/")
	if i < 0 {
		return rawURL + "/"
	}
	return rawURL[:i+1]
}

// findFieldExpr looks up a field by name in a list of AST declarations and
// returns its value expression, or nil if not found.
func findFieldExpr(decls []ast.Decl, name string) ast.Expr {
	for _, d := range decls {
		f, ok := d.(*ast.Field)
		if !ok {
			continue
		}
		if labelStr(f.Label) == name {
			return f.Value
		}
	}
	return nil
}

// findStringField looks up a string-literal field by name and returns its value.
func findStringField(decls []ast.Decl, name string) string {
	expr := findFieldExpr(decls, name)
	if expr == nil {
		return ""
	}
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return lit.Value
	}
	return s
}

// labelStr returns the string value of an AST label (identifier or string literal).
func labelStr(label ast.Label) string {
	switch l := label.(type) {
	case *ast.Ident:
		return l.Name
	case *ast.BasicLit:
		s, err := strconv.Unquote(l.Value)
		if err != nil {
			return l.Value
		}
		return s
	}
	return ""
}
