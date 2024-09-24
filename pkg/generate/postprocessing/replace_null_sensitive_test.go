package postprocessing

import "testing"

func TestReplaceNullSensitiveAttributes(t *testing.T) {
	for _, testFile := range []string{
		"testdata/replace-user-password.tf",
	} {
		postprocessingTest(t, testFile, func(fpath string) {
			ReplaceNullSensitiveAttributes(fpath)
		})
	}
}
