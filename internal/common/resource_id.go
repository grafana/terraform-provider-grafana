package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type ResourceIDFieldType string

var (
	defaultSeparator          = ":"
	ResourceIDFieldTypeInt    = ResourceIDFieldType("int")
	ResourceIDFieldTypeString = ResourceIDFieldType("string")
	allIDs                    = []*ResourceID{}
)

type ResourceIDField struct {
	Name string
	Type ResourceIDFieldType
	// Optional bool // Unimplemented. Will be used for org ID
}

func StringIDField(name string) ResourceIDField {
	return ResourceIDField{
		Name: name,
		Type: ResourceIDFieldTypeString,
	}
}

func IntIDField(name string) ResourceIDField {
	return ResourceIDField{
		Name: name,
		Type: ResourceIDFieldTypeInt,
	}
}

type ResourceID struct {
	resourceName   string
	separators     []string
	expectedFields []ResourceIDField
}

func NewResourceID(resourceName string, expectedFields ...ResourceIDField) *ResourceID {
	return newResourceIDWithSeparators(resourceName, []string{defaultSeparator}, expectedFields...)
}

// Deprecated: Use NewResourceID instead
// We should standardize on a single separator, so that function should only be used for old resources
// On major versions, switch to NewResourceID and remove uses of this function
func NewResourceIDWithLegacySeparator(resourceName, legacySeparator string, expectedFields ...ResourceIDField) *ResourceID {
	return newResourceIDWithSeparators(resourceName, []string{defaultSeparator, legacySeparator}, expectedFields...)
}

func newResourceIDWithSeparators(resourceName string, separators []string, expectedFields ...ResourceIDField) *ResourceID {
	tfID := &ResourceID{
		resourceName:   resourceName,
		separators:     separators,
		expectedFields: expectedFields,
	}
	allIDs = append(allIDs, tfID)
	return tfID
}

func (id *ResourceID) Example() string {
	fields := make([]string, len(id.expectedFields))
	for i := range fields {
		fields[i] = fmt.Sprintf("{{ %s }}", id.expectedFields[i].Name)
	}
	return fmt.Sprintf(`terraform import %s.name %q
`, id.resourceName, strings.Join(fields, defaultSeparator))
}

// Make creates a resource ID from the given parts
// The parts must have the correct number of fields and types
func (id *ResourceID) Make(parts ...any) string {
	if len(parts) != len(id.expectedFields) {
		panic(fmt.Sprintf("expected %d fields, got %d", len(id.expectedFields), len(parts))) // This is a coding error, so panic is appropriate
	}
	stringParts := make([]string, len(parts))
	for i, part := range parts {
		// Unwrap pointers
		if reflect.ValueOf(part).Kind() == reflect.Ptr {
			part = reflect.ValueOf(part).Elem().Interface()
		}
		expectedField := id.expectedFields[i]
		switch expectedField.Type {
		case ResourceIDFieldTypeInt:
			asInt, ok := part.(int64)
			if !ok {
				panic(fmt.Sprintf("expected int64 for field %q, got %T", expectedField.Name, part)) // This is a coding error, so panic is appropriate
			}
			stringParts[i] = strconv.FormatInt(asInt, 10)
		case ResourceIDFieldTypeString:
			asString, ok := part.(string)
			if !ok {
				panic(fmt.Sprintf("expected string for field %q, got %T", expectedField.Name, part)) // This is a coding error, so panic is appropriate
			}
			stringParts[i] = asString
		}
	}

	return strings.Join(stringParts, defaultSeparator)
}

// Single parses a resource ID into a single value
func (id *ResourceID) Single(resourceID string) (any, error) {
	parts, err := id.Split(resourceID)
	if err != nil {
		return nil, err
	}
	return parts[0], nil
}

// Split parses a resource ID into its parts
// The parts will be cast to the expected types
func (id *ResourceID) Split(resourceID string) ([]any, error) {
	for _, sep := range id.separators {
		parts := strings.Split(resourceID, sep)
		if len(parts) == len(id.expectedFields) {
			partsAsAny := make([]any, len(parts))
			for i, part := range parts {
				expectedField := id.expectedFields[i]
				switch expectedField.Type {
				case ResourceIDFieldTypeInt:
					asInt, err := strconv.ParseInt(part, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("expected int for field %q, got %q", expectedField.Name, part)
					}
					partsAsAny[i] = asInt
				case ResourceIDFieldTypeString:
					partsAsAny[i] = part
				}
			}

			return partsAsAny, nil
		}
	}

	expectedFieldNames := make([]string, len(id.expectedFields))
	for i, f := range id.expectedFields {
		expectedFieldNames[i] = f.Name
	}
	return nil, fmt.Errorf("id %q does not match expected format. Should be in the format: %s", resourceID, strings.Join(expectedFieldNames, defaultSeparator))
}

// GenerateImportFiles generates import files for all resources that use a helper defined in this package
func GenerateImportFiles(path string) error {
	for _, id := range allIDs {
		resourcePath := filepath.Join(path, "resources", id.resourceName, "import.sh")
		log.Printf("Generating import file for %s (writing to %s)\n", id.resourceName, resourcePath)
		err := os.WriteFile(resourcePath, []byte(id.Example()), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}
