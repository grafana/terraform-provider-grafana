package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestBlocks(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), "testblocks.tf")
	err := writeBlocks(tempFile,
		providerBlock(map[string]any{
			"auth": "admin:admin", // Supports strings without cty wrapper
			"http_headers": cty.MapVal(map[string]cty.Value{
				"header1": cty.StringVal("val2"),
			}),
			"url": cty.StringVal("hello.com"),
		}),
		resourceBlock("grafana_cloud_stack", "my-stack", map[string]any{
			"region": traversal("data", "region", "slug"),
			"slug":   "hello",
			"other":  cty.ListVal([]cty.Value{cty.StringVal("123"), cty.StringVal("456")}),
			"sub_block": []map[string]any{
				{},
				{
					"attr": "val",
				},
			},
		}),
	)
	require.NoError(t, err)

	gotContent, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	expectedContent, err := os.ReadFile("testdata/testblocks.hcl")
	require.NoError(t, err)
	assert.Equal(t, string(expectedContent), string(gotContent))
}
