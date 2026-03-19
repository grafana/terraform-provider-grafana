package codegen

import (
	"context"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
)

type Config struct {
	// SchemaUrl is the raw URL of the CUE file that defines the kind.
	SchemaUrl string
	// Subpath is the CUE path to the spec within the file (e.g. "checkv0alpha1.schema.spec").
	Subpath string
	// Name is the user-chosen schema name used as the forced_envelope (e.g. "Check").
	Name string
	// KindName is the kind name extracted from the CUE file (e.g. "Check").
	KindName string
	// PluralName is the plural name extracted from the CUE file (e.g. "checks").
	PluralName string
	// Version is the selected API version (e.g. "v0alpha1").
	Version string
	// AppName is the Grafana app name from the manifest (e.g. "advisor").
	AppName string
	// OutputDir is the target directory under internal/resources (e.g. "appplatform").
	OutputDir string
	// SkipFormatting disables gofmt/goimports after generation.
	SkipFormatting bool
}

func Generate(config *Config) error {
	v, err := LoadCueValue(config.SchemaUrl)
	if err != nil {
		return err
	}

	jennies := codejen.JennyListWithNamer[cue.Value](func(_ cue.Value) string {
		return "CueResourceGenerator"
	})

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	templatesDir := filepath.Join(cwd, "codegen", "templates", "appplatform")

	jennies.AppendOneToMany(&GoResourceGenerator{
		name:           config.Name,
		subpath:        config.Subpath,
		kindName:       config.KindName,
		pluralName:     config.PluralName,
		version:        config.Version,
		appName:        config.AppName,
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
