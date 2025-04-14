package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortResources(t *testing.T) {
	t.Parallel()

	goldenFiles, err := filepath.Glob("testdata/sort/*-golden.tf")
	require.NoError(t, err)

	for _, goldenFile := range goldenFiles {
		testFile := strings.Replace(goldenFile, "-golden.tf", ".tf", 1)
		t.Run(testFile, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(testFile)
			require.NoError(t, err)

			sortedContent := sortResources(string(content))

			goldenContent, err := os.ReadFile(goldenFile)
			require.NoError(t, err)

			require.Equal(t, string(goldenContent), sortedContent)
		})
	}
}
