package postprocessing

import "testing"

func TestUsePreferredResourceNames(t *testing.T) {
	for _, testFile := range []string{
		"testdata/preferred-resource-name.tf",
	} {
		postprocessingTest(t, testFile, func(fpath string) {
			UsePreferredResourceNames(fpath)
		})
	}
}
