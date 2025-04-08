package postprocessing

import (
	"fmt"
	"testing"
)

func TestUsePreferredResourceNames(t *testing.T) {
	for _, testFile := range []string{
		"testdata/preferred-resource-name.tf",
	} {
		postprocessingTest(t, testFile, func(fpath string) {
			UsePreferredResourceNames(fpath)
		})
	}
}

func TestCleanResourceName(t *testing.T) {
	var tests = []struct {
		before, after string
	}{
		{"qwerty", "qwerty"},
		{"123", "_123"},
		{"-foo", "_-foo"},
		{"_bar", "_bar"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s", tt.before, tt.after)
		t.Run(testname, func(t *testing.T) {
			cleaned := CleanResourceName(tt.before)
			if cleaned != tt.after {
				t.Errorf(`Resource name %q was clean as %q and not %q as expected`, tt.before, cleaned, tt.after)
			}
		})
	}
}
