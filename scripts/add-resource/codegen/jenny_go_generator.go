package codegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
	"github.com/grafana/cog"
)

var _ codejen.OneToOne[cue.Value] = &GoResourceGenerator{}

type GoResourceGenerator struct {
	name           string
	subpath        string
	kindName       string
	pluralName     string
	version        string
	appName        string
	serviceName    string
	groupOverride  string
	outputDir      string
	templatesDir   string
	skipFormatting bool
}

func (jenny *GoResourceGenerator) JennyName() string {
	return "GoResourceGenerator"
}

func (jenny *GoResourceGenerator) Generate(v cue.Value) (*codejen.File, error) {
	var opts []cog.CUEOption
	if jenny.subpath != "" {
		v = v.LookupPath(cue.ParsePath(jenny.subpath))
		opts = append(opts, cog.ForceEnvelope(fmt.Sprintf("%sSpecModel", jenny.name)))
	} else {
		v = v.LookupPath(cue.ParsePath(jenny.name))
		opts = append(opts, cog.ForceEnvelope(jenny.name))
	}

	// groupOverride may be a Kubernetes API group like "advisor.grafana.app".
	// For the Go import path we only need the first segment before the first dot.
	apiGroup := jenny.appName
	if jenny.groupOverride != "" {
		apiGroup = strings.SplitN(jenny.groupOverride, ".", 2)[0]
	}
	// serviceName is derived from the manifest URL (segment after "apps/") and is the
	// Go module directory name. Falls back to appName if not set.
	serviceName := jenny.serviceName
	if serviceName == "" {
		serviceName = jenny.appName
	}
	importPath := fmt.Sprintf("github.com/grafana/grafana/apps/%s/pkg/apis/%s/%s",
		serviceName, apiGroup, jenny.version)
	modulePath := fmt.Sprintf("github.com/grafana/grafana/apps/%s", serviceName)
	envelopeName := fmt.Sprintf("%sSpecModel", jenny.name)

	// os.DirFS wraps jenny.templatesDir as an fs.FS.
	// ParseFS(fs, "custom") in initTemplates will walk "custom/" within that FS,
	// which maps to <templatesDir>/custom/ on disk.
	customFS := os.DirFS(jenny.templatesDir)

	files, err := cog.TypesFromSchema().
		CUEValue(jenny.outputDir, v, opts...).
		Terraform(cog.TerraformConfig{
			PrefixAttributeSpec:  jenny.name,
			SkipPostFormatting:   jenny.skipFormatting,
			CustomTemplatesFS:    customFS,
			CustomTemplatesFuncs: jenny.buildTemplateFuncs(envelopeName, importPath, modulePath),
		}).
		Run(context.Background())
	if err != nil {
		return nil, err
	}

	outputPath := filepath.Join(jenny.outputDir, fmt.Sprintf("%s_resource.go", strings.ToLower(jenny.name)))
	return codejen.NewFile(outputPath, files[0].Data, jenny), nil
}

// buildTemplateFuncs returns the function map injected into cog's template engine.
// These functions give the `object_all_custom_methods` template access to kind metadata
// and scalar-type helpers for auto-generating SpecParser/SpecSaver field mappings.
//
// The functions accept `any` instead of ast.Type to avoid importing cog's internal packages.
// The ast.Kind and ast.ScalarKind types are string-based, so reflect.Value.String() works.
func (jenny *GoResourceGenerator) buildTemplateFuncs(envelopeName, importPath, modulePath string) map[string]any {
	return map[string]any{
		// Metadata accessors used in the template.
		"envelopeName":   func() string { return envelopeName },
		"schemaName":     func() string { return jenny.name },
		"kindName":       func() string { return jenny.kindName },
		"pluralName":     func() string { return jenny.pluralName },
		"version":        func() string { return jenny.version },
		"kindImportPath": func() string { return importPath },
		"kindModule":     func() string { return modulePath },

		// isSimpleScalar returns true for scalar types (nullable or not) that can be
		// auto-mapped between Terraform SDK types and native Go types.
		"isSimpleScalar": func(t any) bool {
			info, ok := scalarInfoOf(t)
			if !ok {
				return false
			}
			switch info.kind {
			case "string", "bool",
				"int8", "int16", "int32", "int64",
				"uint8", "uint16", "uint32", "uint64",
				"float32", "float64":
				return true
			}
			return false
		},

		// tfValueOf returns the Terraform value-getter method for a scalar field.
		// Nullable types use the pointer variant (e.g. "ValueStringPointer()" for *string).
		"tfValueOf": func(t any) string {
			info, ok := scalarInfoOf(t)
			if !ok {
				return ""
			}
			ptr := ""
			if info.nullable {
				ptr = "Pointer"
			}
			switch info.kind {
			case "string":
				return "ValueString" + ptr + "()"
			case "bool":
				return "ValueBool" + ptr + "()"
			case "float32", "float64":
				return "ValueFloat64" + ptr + "()"
			default: // int / uint variants
				return "ValueInt64" + ptr + "()"
			}
		},

		// tfTypeValueOf returns the Terraform constructor for converting a native Go value
		// to a Terraform SDK type. Nullable types use the pointer variant
		// (e.g. "types.StringPointerValue" for *string).
		"tfTypeValueOf": func(t any) string {
			info, ok := scalarInfoOf(t)
			if !ok {
				return ""
			}
			ptr := ""
			if info.nullable {
				ptr = "Pointer"
			}
			switch info.kind {
			case "string":
				return "types.String" + ptr + "Value"
			case "bool":
				return "types.Bool" + ptr + "Value"
			case "float32", "float64":
				return "types.Float64" + ptr + "Value"
			default:
				return "types.Int64" + ptr + "Value"
			}
		},

		// attrTypeOf returns the attr.Type constant for use in types.ObjectValueFrom maps
		// (e.g. "types.StringType" for string). Nullability does not affect the attr.Type.
		"attrTypeOf": func(t any) string {
			info, ok := scalarInfoOf(t)
			if !ok {
				return ""
			}
			switch info.kind {
			case "string":
				return "types.StringType"
			case "bool":
				return "types.BoolType"
			case "float32", "float64":
				return "types.Float64Type"
			default:
				return "types.Int64Type"
			}
		},
	}
}

type scalarInfo struct {
	kind     string
	nullable bool
}

// scalarInfoOf extracts kind and nullability from an ast.Type value passed as any.
// ast.Kind and ast.ScalarKind are both string-based types, so reflect works directly.
// Returns ({}, false) if t is not a scalar type or is missing the Scalar field.
func scalarInfoOf(t any) (scalarInfo, bool) {
	v := reflect.ValueOf(t)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return scalarInfo{}, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return scalarInfo{}, false
	}

	kindField := v.FieldByName("Kind")
	if !kindField.IsValid() || kindField.String() != "scalar" {
		return scalarInfo{}, false
	}

	nullable := false
	if nf := v.FieldByName("Nullable"); nf.IsValid() {
		nullable = nf.Bool()
	}

	scalarField := v.FieldByName("Scalar")
	if !scalarField.IsValid() || scalarField.IsNil() {
		return scalarInfo{}, false
	}

	scalarKindField := scalarField.Elem().FieldByName("ScalarKind")
	if !scalarKindField.IsValid() {
		return scalarInfo{}, false
	}

	return scalarInfo{kind: scalarKindField.String(), nullable: nullable}, true
}
