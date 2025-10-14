package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
)

func main() {
	docsPath := os.Args[1]
	if err := setResourceCategories(docsPath); err != nil {
		panic(err)
	}
}

func setResourceCategories(docsPath string) error {
	for _, r := range provider.Resources() {
		if err := setResourceCategory(r.Name, string(r.Category), docsPath); err != nil {
			return err
		}
	}

	for _, r := range provider.AppPlatformResources() {
		if err := setResourceCategory(r.Name, string(r.Category), docsPath); err != nil {
			return err
		}
	}

	for _, d := range provider.DataSources() {
		if d.Category == "" {
			return fmt.Errorf("data source %s does not have a category", d.Name)
		}
		name := strings.TrimPrefix(d.Name, "grafana_")
		datasourceFileName := filepath.Join(docsPath, "data-sources", name+".md")
		if err := setCategory(datasourceFileName, string(d.Category)); err != nil {
			return err
		}
	}

	return nil
}

func setResourceCategory(name, category, docsPath string) error {
	if category == "" {
		return fmt.Errorf("resource %s does not have a category", name)
	}
	name = strings.TrimPrefix(name, "grafana_")
	resourceFileName := filepath.Join(docsPath, "resources", name+".md")
	return setCategory(resourceFileName, category)
}

func setCategory(fpath string, category string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read the file
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Set the category
	content := strings.Replace(string(b), `subcategory: ""`, fmt.Sprintf(`subcategory: "%s"`, category), 1)

	// Write the file
	return os.WriteFile(fpath, []byte(content), 0600)
}
