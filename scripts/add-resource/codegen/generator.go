package codegen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/codejen"
)

type Config struct {
	Url            string
	OutputDir      string
	Name           string
	Subpath        string
	SkipFormatting bool
}

func Generate(config *Config) error {
	v, err := loadCueFile(config.Url)
	if err != nil {
		return err
	}

	jennies := codejen.JennyListWithNamer[cue.Value](func(_ cue.Value) string {
		return "CueResourceGenerator"
	})

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	jennies.Append(&GoResourceGenerator{
		name:           config.Name,
		subpath:        config.Subpath,
		outputDir:      filepath.Join("internal", "resources", config.OutputDir),
		skipFormatting: config.SkipFormatting,
	})

	files, err := jennies.GenerateFS(v)
	if err != nil {
		return err
	}

	return files.Write(context.Background(), filepath.Join(cwd, "../.."))
}

func loadCueFile(url string) (cue.Value, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GITHUB_PAT")))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to do request url: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return cue.Value{}, fmt.Errorf("failed to get raw url: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to read body: %w", err)
	}

	v := cuecontext.New().CompileString(string(data))
	if v.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to compile cue file: %w", v.Err())
	}

	return v, nil
}
