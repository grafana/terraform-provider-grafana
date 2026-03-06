package codegen

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grafana/codejen"
	"github.com/grafana/cog"
)

var _ codejen.OneToOne[cue.Value] = &GoResourceGenerator{}

type GoResourceGenerator struct {
	name           string
	subpath        string
	outputDir      string
	pkg            string
	skipFormatting bool
}

func (jenny *GoResourceGenerator) JennyName() string {
	return "GoResourceGenerator"
}

func (jenny *GoResourceGenerator) Generate(v cue.Value) (*codejen.File, error) {
	var opts []cog.CUEOption
	if jenny.subpath != "" {
		opts = append(opts, cog.ForceEnvelope(fmt.Sprintf("%sSpecModel", jenny.name)))
	}

	files, err := cog.TypesFromSchema().
		CUEValue(jenny.outputDir, v.LookupPath(cue.ParsePath(jenny.subpath)), opts...).
		Terraform(cog.TerraformConfig{
			PrefixAttributeSpec: jenny.name,
			SkipPostFormatting:  jenny.skipFormatting,
		}).
		Run(context.Background())

	if err != nil {
		return nil, err
	}

	path := filepath.Join(jenny.outputDir, fmt.Sprintf("%s_resource.go", strings.ToLower(jenny.name)))
	file := codejen.NewFile(path, files[0].Data, jenny)
	return file, nil
}
