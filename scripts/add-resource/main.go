package main

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/grafana/terraform-provider-grafana/scripts/add-resource/codegen"
)

type resourceDescriptor struct {
	SchemaUrl string
	OutputDir string
	Subpath   string
	Spec      string
	Envelope  string
}

func notEmpty() func(s string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("required.")
		}
		return nil
	}
}

func main() {
	descriptor := resourceDescriptor{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Schema URL").
				Description("Raw URL from the cue schema. Example: https://raw.githubusercontent.com/grafana/grafana/refs/heads/main/apps/folder/kinds/folder.cue").
				Validate(func(str string) error {
					u, err := url.ParseRequestURI(str)
					if err != nil {
						return err
					}
					filename := path.Base(u.Path)
					if strings.HasSuffix(filename, ".cue") {
						return nil
					}

					return fmt.Errorf("must be a .cue file, got %q", filename)
				}).
				Value(&descriptor.SchemaUrl),

			huh.NewInput().
				Title("Output directory").
				Description("Directory where the generated code will be placed. Example: appplatform.").
				Placeholder("appplatform").
				Validate(notEmpty()).
				Value(&descriptor.OutputDir),
			huh.NewSelect[string]().
				Title("Schema definition").
				Description("Select the schema to use. If the schema is under a path other than the root, select 'subpath'.").
				Options(
					huh.NewOption("Subpath", "subpath"),
					huh.NewOption("Spec", "spec"),
				).
				Value(&descriptor.Spec),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Subpath").
				Description("Path to the schema definition. Example: foldersV1beta1.schema.spec").
				Validate(notEmpty()).
				Value(&descriptor.Subpath),
			huh.NewInput().
				Title("Envelope name for subpath").
				Description("Name of the resource to envelope the spec values. Example: Folder").
				Validate(notEmpty()).
				Value(&descriptor.Envelope),
		).WithHideFunc(func() bool {
			return descriptor.Spec == "spec"
		}),
		huh.NewGroup(
			huh.NewInput().
				Title("Spec").
				Description("Write the main definition to use as spec. Example DashboardSpec").
				Validate(notEmpty()).
				Value(&descriptor.Envelope),
		).WithHideFunc(func() bool {
			return descriptor.Spec == "subpath"
		}),
	)

	if err := form.Run(); err != nil {
		panic(err)
	}

	if err := codegen.Generate(&codegen.Config{
		Url:            descriptor.SchemaUrl,
		OutputDir:      descriptor.OutputDir,
		Name:           descriptor.Envelope,
		Subpath:        descriptor.Subpath,
		SkipFormatting: false,
	}); err != nil {
		panic(err)
	}
}
