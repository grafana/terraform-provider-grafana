package codegen

import (
	"context"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
)

type Config struct {
	// PackageValue is the pre-compiled CUE package value from FetchAndParseManifest.
	// If set, SchemaUrl is not used for loading.
	PackageValue *cue.Value
	// SchemaUrl is the raw URL of the CUE file that defines the kind.
	// Used only when PackageValue is nil.
	SchemaUrl string
	// Subpath is the CUE path to the spec within the package (e.g. "checkv0alpha1.schema.spec").
	Subpath string
	// Name is the user-chosen schema name used as the forced_envelope (e.g. "Check").
	Name string
	// KindName is the kind name extracted from the CUE file (e.g. "Check").
	KindName string
	// PluralName is the plural name extracted from the CUE file (e.g. "checks").
	PluralName string
	// Version is the selected API version (e.g. "v0alpha1").
	Version string
	// AppName is the Grafana app module name from the manifest (e.g. "correlations").
	AppName string
	// ServiceName is the Go module directory derived from the manifest URL path (e.g. "iam" from "apps/iam/kinds/manifest.cue").
	// Used as the first segment of the Go import path. Falls back to AppName if empty.
	ServiceName string
	// GroupOverride is the API group name when it differs from AppName (e.g. "correlation").
	// If empty, AppName is used for the API group path.
	GroupOverride string
	// OutputDir is the target directory under internal/resources (e.g. "appplatform").
	OutputDir string
	// SkipFormatting disables gofmt/goimports after generation.
	SkipFormatting bool
}

func Generate(config *Config) error {
	var v cue.Value
	if config.PackageValue != nil {
		v = *config.PackageValue
	} else {
		var err error
		v, err = LoadCueValue(config.SchemaUrl)
		if err != nil {
			return err
		}
	}

	jennies := codejen.JennyListWithNamer[cue.Value](func(_ cue.Value) string {
		return "CueResourceGenerator"
	})

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	templatesDir := filepath.Join(cwd, "codegen", "templates")

	jennies.Append(&GoResourceGenerator{
		name:           config.Name,
		subpath:        config.Subpath,
		kindName:       config.KindName,
		pluralName:     config.PluralName,
		version:        config.Version,
		appName:        config.AppName,
		serviceName:    config.ServiceName,
		groupOverride:  config.GroupOverride,
		outputDir:      filepath.Join("internal", "resources", config.OutputDir),
		skipFormatting: config.SkipFormatting,
		templatesDir:   templatesDir,
	})

	files, err := jennies.GenerateFS(v)
	if err != nil {
		return err
	}

	return files.Write(context.Background(), filepath.Join(cwd, "../.."))
}
