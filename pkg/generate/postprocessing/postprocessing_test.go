package postprocessing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func postprocessingTest(t *testing.T, testFile string, fn func(fpath string)) {
	t.Helper()

	t.Run(testFile, func(t *testing.T) {
		goldenFilepath := strings.Replace(testFile, ".tf", ".golden.tf", 1)

		// Copy the file to a temporary location
		tmpFilepath := filepath.Join(t.TempDir(), filepath.Base(testFile))
		file, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFilepath, file, 0600))

		// Run the postprocessing function
		fn(tmpFilepath)

		// Compare the file with the golden file
		got, err := os.ReadFile(tmpFilepath)
		require.NoError(t, err)
		want, err := os.ReadFile(goldenFilepath)
		require.NoError(t, err)

		require.Equal(t, string(want), string(got))
	})
}
