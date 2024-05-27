package generate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v2/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v2/pkg/generate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestAccGenerate_Dashboard(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
				Check: func(s *terraform.State) error {
					tempDir := t.TempDir()
					config := generate.Config{
						OutputDir:       tempDir,
						Clobber:         true,
						Format:          generate.OutputFormatHCL,
						ProviderVersion: "v3.0.0",
						Grafana: &generate.GrafanaConfig{
							URL:  "http://localhost:3000",
							Auth: "admin:admin",
						},
					}

					require.NoError(t, generate.Generate(context.Background(), &config))
					assertFiles(t, tempDir, "testdata/generate/dashboard-expected", "", []string{
						".terraform",
						".terraform.lock.hcl",
					})

					return nil
				},
			},
		},
	})
}

// assertFiles checks that all files in the "expectedFilesDir" directory match the files in the "gotFilesDir" directory.
func assertFiles(t *testing.T, gotFilesDir, expectedFilesDir, subdir string, ignoreDirEntries []string) {
	t.Helper()

	originalGotFilesDir := gotFilesDir
	originalExpectedFilesDir := expectedFilesDir
	if subdir != "" {
		gotFilesDir = filepath.Join(gotFilesDir, subdir)
		expectedFilesDir = filepath.Join(expectedFilesDir, subdir)
	}

	// Check that all generated files are expected (recursively)
	gotFiles, err := os.ReadDir(gotFilesDir)
	if err != nil {
		t.Logf("folder %s was not generated as expected", subdir)
		t.Fail()
		return
	}
	for _, gotFile := range gotFiles {
		relativeName := filepath.Join(subdir, gotFile.Name())
		if slices.Contains(ignoreDirEntries, relativeName) {
			continue
		}

		if gotFile.IsDir() {
			assertFiles(t, originalGotFilesDir, originalExpectedFilesDir, filepath.Join(subdir, gotFile.Name()), ignoreDirEntries)
			continue
		}

		if _, err := os.Stat(filepath.Join(expectedFilesDir, gotFile.Name())); err != nil {
			t.Logf("file %s was generated but wasn't expected", relativeName)
			t.Fail()
		}
	}

	// Verify the contents of the generated files (recursively)
	// All files in the expected directory should be present in the generated directory
	expectedFiles, err := os.ReadDir(expectedFilesDir)
	if err != nil {
		t.Logf("folder %s was generated but wasn't expected", subdir)
		t.Fail()
		return
	}
	for _, expectedFile := range expectedFiles {
		if expectedFile.IsDir() {
			assertFiles(t, originalGotFilesDir, originalExpectedFilesDir, filepath.Join(subdir, expectedFile.Name()), ignoreDirEntries)
			continue
		}
		expectedContent, err := os.ReadFile(filepath.Join(expectedFilesDir, expectedFile.Name()))
		require.NoError(t, err)

		gotContent, err := os.ReadFile(filepath.Join(gotFilesDir, expectedFile.Name()))
		require.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(string(expectedContent)), strings.TrimSpace(string(gotContent)))
	}
}
