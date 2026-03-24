package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
)

// ManifestInfo holds extracted metadata from a Grafana manifest.cue.
type ManifestInfo struct {
	// AppName is the app/group name declared in the manifest (e.g. "iam").
	AppName string
	// ServiceName is the Go module directory name, derived from the manifest URL
	// path after "apps/" (e.g. "iam" from "apps/iam/kinds/manifest.cue").
	ServiceName   string
	GroupOverride string
	// Versions maps version names (e.g. "v0alpha1") to kind NAME slices (e.g. ["RoleBinding"]).
	Versions map[string][]string
	// BaseURL is the directory URL for sibling kind files.
	BaseURL string
	// RawURL is the original manifest URL.
	RawURL string
	// PackageValue is the compiled CUE value of the whole kinds package.
	PackageValue cue.Value
	// PackageFiles maps filename to raw content for identifier lookup.
	PackageFiles map[string][]byte
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
	// Identifier is the CUE field label in the package (e.g. "rolebindingv0alpha1").
	Identifier string
	// KindName is the string value of the "kind" field (e.g. "RoleBinding").
	KindName string
	// PluralName is the string value of the "pluralName" field (e.g. "rolebindings").
	PluralName string
	// SpecSubpath is the CUE path to the spec value (e.g. "rolebindingv0alpha1.schema.spec").
	SpecSubpath string
	// FileURL is the URL of the CUE file that defines this kind.
	FileURL string
}

// FetchAndParseManifest compiles the kinds package at rawURL and extracts
// app metadata and versions/kinds using the CUE value API.
func FetchAndParseManifest(rawURL string) (*ManifestInfo, error) {
	pkgVal, files, err := compileKindsPackage(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile kinds package: %w", err)
	}

	info := &ManifestInfo{
		Versions:     make(map[string][]string),
		RawURL:       rawURL,
		BaseURL:      dirURL(rawURL),
		ServiceName:  serviceNameFromURL(rawURL),
		PackageValue: pkgVal,
		PackageFiles: files,
	}

	// Support both "manifest" and "_manifest" field names.
	manifestVal := pkgVal.LookupPath(cue.MakePath(cue.Str("manifest")))
	if !manifestVal.Exists() {
		manifestVal = pkgVal.LookupPath(cue.MakePath(cue.Str("_manifest")))
	}
	if !manifestVal.Exists() {
		return nil, fmt.Errorf("manifest field not found in CUE package")
	}

	// Support both "appName" (newer) and "appId" (legacy).
	if appName, err := manifestVal.LookupPath(cue.MakePath(cue.Str("appName"))).String(); err == nil {
		info.AppName = appName
	} else if appID, err := manifestVal.LookupPath(cue.MakePath(cue.Str("appId"))).String(); err == nil {
		info.AppName = appID
	}

	if groupOverride, err := manifestVal.LookupPath(cue.MakePath(cue.Str("groupOverride"))).String(); err == nil {
		info.GroupOverride = groupOverride
	}

	// Extract versions using the CUE value API.
	versionsVal := manifestVal.LookupPath(cue.MakePath(cue.Str("versions")))
	if versionsVal.Exists() {
		it, err := versionsVal.Fields()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate versions: %w", err)
		}
		for it.Next() {
			versionName := strings.Trim(it.Selector().String(), "\"")
			kindsVal := it.Value().LookupPath(cue.MakePath(cue.Str("kinds")))
			kinds, err := extractKindNames(kindsVal)
			if err != nil {
				return nil, fmt.Errorf("failed to extract kinds for version %s: %w", versionName, err)
			}
			if len(kinds) > 0 {
				info.Versions[versionName] = kinds
			}
		}
	}

	// Legacy format: single "version" string + flat "kinds" list at the manifest level.
	if len(info.Versions) == 0 {
		if version, err := manifestVal.LookupPath(cue.MakePath(cue.Str("version"))).String(); err == nil && version != "" {
			kindsVal := manifestVal.LookupPath(cue.MakePath(cue.Str("kinds")))
			kinds, _ := extractKindNames(kindsVal)
			info.Versions[version] = kinds
		}
	}

	if len(info.Versions) == 0 {
		return nil, fmt.Errorf("no versions found in manifest")
	}

	return info, nil
}

// extractKindNames iterates a CUE list and extracts the "kind" string from each element.
func extractKindNames(kindsVal cue.Value) ([]string, error) {
	if !kindsVal.Exists() {
		return nil, nil
	}
	it, err := kindsVal.List()
	if err != nil {
		return nil, err
	}
	var kinds []string
	for it.Next() {
		kindName, err := it.Value().LookupPath(cue.MakePath(cue.Str("kind"))).String()
		if err != nil || kindName == "" {
			continue
		}
		kinds = append(kinds, kindName)
	}
	return kinds, nil
}

// FetchKindInfo finds the CUE identifier and file URL for a kind by name,
// using the pre-compiled package and raw files from the manifest.
func FetchKindInfo(kindName string, manifest *ManifestInfo) (*KindInfo, error) {
	identifier, pluralName, err := findIdentifierForKind(manifest.PackageValue, kindName)
	if err != nil {
		return nil, fmt.Errorf("kind %q not found in package: %w", kindName, err)
	}

	fileName := findFileForIdentifier(identifier, manifest.PackageFiles)
	fileURL := manifest.BaseURL + fileName

	return &KindInfo{
		Identifier:  identifier,
		KindName:    kindName,
		PluralName:  pluralName,
		SpecSubpath: identifier + ".schema.spec",
		FileURL:     fileURL,
	}, nil
}

// findIdentifierForKind iterates the top-level fields of a compiled CUE package
// and returns the field label whose "kind" field equals kindName.
func findIdentifierForKind(pkgVal cue.Value, kindName string) (identifier, pluralName string, err error) {
	it, err := pkgVal.Fields()
	if err != nil {
		return "", "", fmt.Errorf("iterate package fields: %w", err)
	}
	for it.Next() {
		kn, kerr := it.Value().LookupPath(cue.MakePath(cue.Str("kind"))).String()
		if kerr != nil || kn != kindName {
			continue
		}
		pn, _ := it.Value().LookupPath(cue.MakePath(cue.Str("plural"))).String()
	if pn == "" {
		pn, _ = it.Value().LookupPath(cue.MakePath(cue.Str("pluralName"))).String()
	}
		return it.Selector().String(), pn, nil
	}
	return "", "", fmt.Errorf("no field with kind=%q found", kindName)
}

// findFileForIdentifier scans raw CUE files for the one that declares identifier
// as a top-level field. Returns the filename, or "" if not found.
func findFileForIdentifier(identifier string, files map[string][]byte) string {
	for name, content := range files {
		f, err := parser.ParseFile(name, content, parser.ParseComments)
		if err != nil {
			continue
		}
		if findFieldExpr(f.Decls, identifier) != nil {
			return name
		}
	}
	return ""
}

// compileKindsPackage fetches all .cue files from the kinds directory at rawURL,
// resolves imports via the GitHub API, and compiles the directory as a CUE package.
// Returns the compiled value and a map of filename -> raw content.
func compileKindsPackage(rawURL string) (cue.Value, map[string][]byte, error) {
	ghInfo := parseRawGitHubURL(rawURL)
	if ghInfo == nil {
		return cue.Value{}, nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	dirPath := repoDirPath(rawURL)
	dirFiles, err := fetchGitHubDirectory(ghInfo.owner, ghInfo.repo, dirPath, ghInfo.ref)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("fetch kinds directory: %w", err)
	}

	overlay := cueModuleOverlay()
	allFiles := make(map[string][]byte, len(dirFiles))
	fetched := make(map[string]bool)

	for name, content := range dirFiles {
		overlay["/cog/vfs/kinds/"+name] = load.FromBytes(content)
		allFiles[name] = content

		// Resolve github.com/grafana/grafana/... imports from each file.
		f, err := parser.ParseFile(name, content, parser.ParseComments)
		if err != nil {
			continue
		}
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(importPath, "github.com/grafana/grafana/") || fetched[importPath] {
				continue
			}
			fetched[importPath] = true
			repoPath := strings.TrimPrefix(importPath, "github.com/grafana/grafana/")
			pkgFiles, err := fetchGitHubDirectory(ghInfo.owner, ghInfo.repo, repoPath, ghInfo.ref)
			if err != nil {
				continue
			}
			for fname, fcontent := range pkgFiles {
				overlayPath := filepath.Join("/cog/vfs/cue.mod/pkg", importPath, fname)
				overlay[overlayPath] = load.FromBytes(fcontent)
			}
		}
	}

	bis := load.Instances([]string{"."}, &load.Config{
		Overlay: overlay,
		Dir:     "/cog/vfs/kinds",
	})
	if len(bis) == 0 {
		return cue.Value{}, nil, fmt.Errorf("no CUE instances found")
	}
	if bis[0].Err != nil {
		return cue.Value{}, nil, fmt.Errorf("load CUE package: %w", bis[0].Err)
	}

	value := cuecontext.New().BuildInstance(bis[0])
	if value.Err() != nil {
		return cue.Value{}, nil, fmt.Errorf("build CUE package: %w", value.Err())
	}
	return value, allFiles, nil
}

// repoDirPath extracts the repo-relative directory path from a raw GitHub URL.
// For "https://raw.githubusercontent.com/owner/repo/refs/heads/main/apps/iam/kinds/manifest.cue"
// it returns "apps/iam/kinds".
func repoDirPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "raw.githubusercontent.com" {
		return ""
	}
	ghInfo := parseRawGitHubURL(rawURL)
	if ghInfo == nil {
		return ""
	}
	refParts := strings.Split(ghInfo.ref, "/")
	// allParts: [owner, repo, ref_part1, ..., ref_partN, path_part1, ..., filename]
	allParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	skip := 2 + len(refParts) // skip owner + repo + ref segments
	if skip >= len(allParts) {
		return ""
	}
	pathParts := allParts[skip : len(allParts)-1] // exclude filename
	return strings.Join(pathParts, "/")
}

// rawGitHubURLInfo holds components extracted from a raw.githubusercontent.com URL.
type rawGitHubURLInfo struct {
	owner string
	repo  string
	ref   string
}

// parseRawGitHubURL extracts owner, repo, and ref from a raw GitHub URL.
// Handles both "refs/heads/main" and bare branch names.
func parseRawGitHubURL(rawURL string) *rawGitHubURLInfo {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "raw.githubusercontent.com" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 4 {
		return nil
	}
	owner, repo := parts[0], parts[1]
	var ref string
	if parts[2] == "refs" && len(parts) > 4 {
		ref = strings.Join(parts[2:5], "/")
	} else {
		ref = parts[2]
	}
	return &rawGitHubURLInfo{owner: owner, repo: repo, ref: ref}
}

// fetchGitHubDirectory fetches all *.cue files in a repository directory via
// the GitHub Contents API. Returns a map of filename → file content.
func fetchGitHubDirectory(owner, repo, dirPath, ref string) (map[string][]byte, error) {
	// Note: ref may contain slashes (e.g. "refs/heads/main") which must NOT be
	// percent-encoded in the query string — the GitHub API expects them raw.
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, dirPath, ref)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if pat := os.Getenv("GITHUB_PAT"); pat != "" {
		req.Header.Set("Authorization", "Bearer "+pat)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %s: HTTP %s", apiURL, resp.Status)
	}

	var entries []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode GitHub API response: %w", err)
	}

	files := make(map[string][]byte)
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".cue") {
			continue
		}
		data, err := FetchURL(entry.DownloadURL)
		if err != nil {
			continue
		}
		files[entry.Name] = data
	}
	return files, nil
}

// cueModuleOverlay returns the minimal CUE overlay entry required by cue/load:
// a module.cue declaring the virtual module path used for the overlay FS.
func cueModuleOverlay() map[string]load.Source {
	return map[string]load.Source{
		"/cog/vfs/cue.mod/module.cue": load.FromBytes([]byte(
			"language: { version: \"v0.10.1\" }\nmodule: \"cog.vfs\"\n",
		)),
	}
}

// LoadCueValue fetches a CUE file from a URL and compiles it to a cue.Value.
// Any github.com/grafana/grafana/... imports are resolved by fetching the
// corresponding directory from GitHub via the Contents API.
func LoadCueValue(fileURL string) (cue.Value, error) {
	data, err := FetchURL(fileURL)
	if err != nil {
		return cue.Value{}, err
	}

	f, err := parser.ParseFile("kind.cue", data, parser.ParseComments)
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to parse CUE: %w", err)
	}

	pkgName := f.PackageName()
	if pkgName == "" {
		pkgName = "main"
	}

	overlay := cueModuleOverlay()
	overlay["/cog/vfs/cue.mod/pkg/github.com/cog-vfs/"+pkgName+"/kind.cue"] = load.FromBytes(data)

	ghInfo := parseRawGitHubURL(fileURL)
	fetched := make(map[string]bool)
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if ghInfo == nil || !strings.HasPrefix(importPath, "github.com/grafana/grafana/") || fetched[importPath] {
			continue
		}
		fetched[importPath] = true
		repoPath := strings.TrimPrefix(importPath, "github.com/grafana/grafana/")
		pkgFiles, err := fetchGitHubDirectory(ghInfo.owner, ghInfo.repo, repoPath, ghInfo.ref)
		if err != nil {
			continue
		}
		for name, content := range pkgFiles {
			overlayPath := filepath.Join("/cog/vfs/cue.mod/pkg", importPath, name)
			overlay[overlayPath] = load.FromBytes(content)
		}
	}

	bis := load.Instances([]string{"github.com/cog-vfs/" + pkgName}, &load.Config{
		Overlay: overlay,
		Dir:     "/cog/vfs",
	})
	value := cuecontext.New().BuildInstance(bis[0])
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to compile CUE: %w", value.Err())
	}
	return value, nil
}

// FetchURL performs an HTTP GET request, using GITHUB_PAT env var if set.
func FetchURL(rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if pat := os.Getenv("GITHUB_PAT"); pat != "" {
		req.Header.Set("Authorization", "Bearer "+pat)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %s", rawURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// serviceNameFromURL extracts the Go module directory name from a manifest URL.
// For "…/apps/iam/kinds/manifest.cue" it returns "iam".
func serviceNameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "apps" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
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
