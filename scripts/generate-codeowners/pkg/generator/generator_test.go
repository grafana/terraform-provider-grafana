package generator_test

import (
	"os"
	"testing"

	"github.com/grafanalabs/terraform-provider-grafana/scripts/generate-codeowners/pkg/generator"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name           string
		wantErr        bool
		repoRoot       string
		codeownersFile *os.File
		pathsToCheck   []string
	}{
		{
			name:           "empty paths",
			pathsToCheck:   []string{},
			wantErr:        false,
			repoRoot:       ".",
			codeownersFile: os.Stdout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := generator.New(tt.repoRoot, tt.codeownersFile)
			err := generator.Generate(tt.pathsToCheck)
			assert.NoError(t, err)
		})
	}
}
