package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var allResources = []*Resource{}

type Resource struct {
	Name                  string
	IDType                *ResourceID
	Schema                *schema.Resource
	PluginFrameworkSchema resource.ResourceWithConfigure
}

func NewResource(name string, idType *ResourceID, schema *schema.Resource) *Resource {
	r := &Resource{
		Name:   name,
		IDType: idType,
		Schema: schema,
	}
	allResources = append(allResources, r)
	return r
}

func NewPluginFrameworkResource(name string, idType *ResourceID, schema resource.ResourceWithConfigure) *Resource {
	r := &Resource{
		Name:                  name,
		IDType:                idType,
		PluginFrameworkSchema: schema,
	}
	allResources = append(allResources, r)
	return r
}

func (r *Resource) ImportExample() string {
	id := r.IDType
	fields := make([]string, len(id.expectedFields))
	for i := range fields {
		fields[i] = fmt.Sprintf("{{ %s }}", id.expectedFields[i].Name)
	}
	return fmt.Sprintf(`terraform import %s.name %q
`, r.Name, strings.Join(fields, defaultSeparator))
}

// GenerateImportFiles generates import files for all resources that use a helper defined in this package
func GenerateImportFiles(path string) error {
	for _, r := range allResources {
		resourcePath := filepath.Join(path, "resources", r.Name, "import.sh")
		if err := os.RemoveAll(resourcePath); err != nil { // Remove the file if it exists
			return err
		}

		if r.IDType == nil {
			log.Printf("Skipping import file generation for %s because it does not have an ID type\n", r.Name)
			continue
		}

		log.Printf("Generating import file for %s (writing to %s)\n", r.Name, resourcePath)
		if err := os.WriteFile(resourcePath, []byte(r.ImportExample()), 0600); err != nil {
			return err
		}
	}
	return nil
}
