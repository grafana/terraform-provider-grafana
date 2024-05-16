package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTFJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testblocks.tf")
	testFileContent, err := os.ReadFile("testdata/testblocks.tf")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(testFile, testFileContent, 0600))
	require.NoError(t, convertToTFJSON(tempDir))

	gotDir, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, gotDir, 1) // Only the JSON file

	gotContent, err := os.ReadFile(testFile + ".json")
	require.NoError(t, err)

	expectedContent, err := os.ReadFile("testdata/testblocks.tf.json")
	require.NoError(t, err)

	assert.Equal(t, string(expectedContent), string(gotContent))
}
