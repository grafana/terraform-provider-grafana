package codegen

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
	"github.com/grafana/cog"
)

var _ codejen.OneToMany[cue.Value] = &GoResourceGenerator{}

type GoResourceGenerator struct {
	name           string
	subpath        string
	kindName       string
	pluralName     string
	version        string
	appName        string
	outputDir      string
	templatesDir   string
	skipFormatting bool
}

func (jenny *GoResourceGenerator) JennyName() string {
	return "GoResourceGenerator"
}

func (jenny *GoResourceGenerator) Generate(v cue.Value) (codejen.Files, error) {
	var opts []cog.CUEOption
	if jenny.subpath != "" {
		v = v.LookupPath(cue.ParsePath(jenny.subpath))
		opts = append(opts, cog.ForceEnvelope(fmt.Sprintf("%sSpecModel", jenny.name)))
	} else {
		v = v.LookupPath(cue.ParsePath(jenny.name))
		opts = append(opts, cog.ForceEnvelope(jenny.name))
	}

	cogFiles, err := cog.TypesFromSchema().
		CUEValue(jenny.outputDir, v, opts...).
		Terraform(cog.TerraformConfig{
			PrefixAttributeSpec: jenny.name,
			SkipPostFormatting:  jenny.skipFormatting,
		}).
		Run(context.Background())
	if err != nil {
		return nil, err
	}

	// File 1: cog-generated types and attributes.
	typesPath := filepath.Join(jenny.outputDir, fmt.Sprintf("%s_types_gen.go", strings.ToLower(jenny.name)))
	typesFile := *codejen.NewFile(typesPath, cogFiles[0].Data, jenny)

	// File 2: resource scaffold with package declaration and imports.
	scaffold, err := jenny.renderScaffold()
	if err != nil {
		return nil, fmt.Errorf("failed to render resource scaffold: %w", err)
	}
	if !jenny.skipFormatting {
		if formatted, fmtErr := format.Source(scaffold); fmtErr == nil {
			scaffold = formatted
		}
	}
	resourcePath := filepath.Join(jenny.outputDir, fmt.Sprintf("%s_resource.go", strings.ToLower(jenny.name)))
	resourceFile := *codejen.NewFile(resourcePath, scaffold, jenny)

	return codejen.Files{typesFile, resourceFile}, nil
}

// scaffoldData is passed to app_platform_resource.tmpl.
type scaffoldData struct {
	// PackageName is the Go package name (e.g. "appplatform").
	PackageName string
	// Name is the user-chosen schema name (e.g. "Check").
	Name string
	// KindName is the kind name from the CUE file (e.g. "Check").
	KindName string
	// PluralName is the plural name from the CUE file (e.g. "checks").
	PluralName string
	// Version is the API version (e.g. "v0alpha1").
	Version string
	// AppName is the Grafana app name (e.g. "advisor").
	AppName string
	// ImportPath is the Go import path for the kind package.
	ImportPath string
}

func (jenny *GoResourceGenerator) renderScaffold() ([]byte, error) {
	tmplPath := filepath.Join(jenny.templatesDir, "app_platform_resource.tmpl")
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New("scaffold").Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	importPath := fmt.Sprintf("github.com/grafana/grafana/apps/%s/pkg/apis/%s/%s",
		jenny.appName, jenny.appName, jenny.version)

	// Package name is the last path component, lower-cased (mirrors cog's formatPackageName).
	pkgName := strings.ToLower(filepath.Base(jenny.outputDir))

	data := scaffoldData{
		PackageName: pkgName,
		Name:        jenny.name,
		KindName:    jenny.kindName,
		PluralName:  jenny.pluralName,
		Version:     jenny.version,
		AppName:     jenny.appName,
		ImportPath:  importPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}
