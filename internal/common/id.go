package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	defaultSeparator = ":"
	allIDs           = []*TFID{}
)

type TFID struct {
	resourceName   string
	separators     []string
	expectedFields []string
}

func NewTFID(resourceName string, expectedFields ...string) *TFID {
	return newTFIDWithSeparators(resourceName, []string{defaultSeparator}, expectedFields...)
}

// Deprecated: Use NewTFID instead
// We should standardize on a single separator, so that function should only be used for old resources
// On major versions, switch to NewTFID and remove uses of this function
func NewTFIDWithLegacySeparator(resourceName, legacySeparator string, expectedFields ...string) *TFID {
	return newTFIDWithSeparators(resourceName, []string{defaultSeparator, legacySeparator}, expectedFields...)
}

func newTFIDWithSeparators(resourceName string, separators []string, expectedFields ...string) *TFID {
	tfID := &TFID{
		resourceName:   resourceName,
		separators:     separators,
		expectedFields: expectedFields,
	}
	allIDs = append(allIDs, tfID)
	return tfID
}

func (id *TFID) Example() string {
	fields := make([]string, len(id.expectedFields))
	for i := range fields {
		fields[i] = fmt.Sprintf("{{ %s }}", id.expectedFields[i])
	}
	return fmt.Sprintf(`terraform import %s.name %q
`, id.resourceName, strings.Join(fields, defaultSeparator))
}

func (id *TFID) Make(parts ...any) string {
	if len(parts) != len(id.expectedFields) {
		panic(fmt.Sprintf("expected %d fields, got %d", len(id.expectedFields), len(parts))) // This is a coding error, so panic is appropriate
	}
	stringParts := make([]string, len(parts))
	for i, part := range parts {
		stringParts[i] = fmt.Sprintf("%v", part)
	}
	return strings.Join(stringParts, defaultSeparator)
}

func (s *TFID) Split(id string) ([]string, error) {
	for _, sep := range s.separators {
		parts := strings.Split(id, sep)
		if len(parts) == len(s.expectedFields) {
			return parts, nil
		}
	}
	return nil, fmt.Errorf("id %q does not match expected format. Should be in the format: %s", id, strings.Join(s.expectedFields, defaultSeparator))
}

// GenerateImportFiles generates import files for all resources that use a helper defined in this package
func GenerateImportFiles(path string) error {
	for _, id := range allIDs {
		resourcePath := filepath.Join(path, "resources", id.resourceName, "import.sh")
		log.Printf("Generating import file for %s (writing to %s)\n", id.resourceName, resourcePath)
		err := os.WriteFile(resourcePath, []byte(id.Example()), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
