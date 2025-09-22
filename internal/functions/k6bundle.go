package functions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &K6BundleFunction{}

type K6BundleFunction struct{}

func NewK6BundleFunction() function.Function {
	return &K6BundleFunction{}
}

func (f *K6BundleFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "k6bundle"
}

func (f *K6BundleFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Bundle multi-file JavaScript/TypeScript k6 tests",
		Description: "Takes a file path to a JavaScript or TypeScript k6 test and bundles it using ESbuild. Returns the bundled JavaScript code as a string.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "file_path",
				Description: "Path to the JavaScript or TypeScript file to bundle",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *K6BundleFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var filePath string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &filePath))
	if resp.Error != nil {
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		resp.Error = function.ConcatFuncErrors(resp.Error,
			function.NewFuncError(fmt.Sprintf("File does not exist: %s", filePath)))
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error,
			function.NewFuncError(fmt.Sprintf("Failed to get absolute path: %v", err)))
		return
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{absPath},
		Bundle:      true,
		Platform:    api.PlatformNode,
		Target:      api.ES2017,
		Format:      api.FormatCommonJS,
		External: []string{
			"k6",
			"k6/*",
			"https",
			"https/*",
		},
		Write: false, // Don't write to disk, return the content
	})

	if len(result.Errors) > 0 {
		var errorMsg string
		for _, buildErr := range result.Errors {
			errorMsg += fmt.Sprintf("ESbuild error: %s\n", buildErr.Text)
		}
		resp.Error = function.ConcatFuncErrors(resp.Error,
			function.NewFuncError(errorMsg))
		return
	}

	if len(result.OutputFiles) == 0 {
		resp.Error = function.ConcatFuncErrors(resp.Error,
			function.NewFuncError("ESbuild produced no output"))
		return
	}

	bundledCode := string(result.OutputFiles[0].Contents)

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, bundledCode))
}
