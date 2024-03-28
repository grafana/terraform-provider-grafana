package common

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type ResourceIDFieldType string

const (
	defaultSeparator          = ":"
	ResourceIDFieldTypeInt    = ResourceIDFieldType("int")
	ResourceIDFieldTypeString = ResourceIDFieldType("string")
)

type ResourceIDField struct {
	Name     string
	Type     ResourceIDFieldType
	Optional bool
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

func OptionalStringIDField(name string) ResourceIDField {
	return ResourceIDField{
		Name:     name,
		Type:     ResourceIDFieldTypeString,
		Optional: true,
	}
}

func OptionalIntIDField(name string) ResourceIDField {
	return ResourceIDField{
		Name:     name,
		Type:     ResourceIDFieldTypeInt,
		Optional: true,
	}
}

type ResourceID struct {
	separators     []string
	expectedFields []ResourceIDField
}

func (id *ResourceID) RequiredFields() []ResourceIDField {
	requiredFields := []ResourceIDField{}
	for _, f := range id.expectedFields {
		if !f.Optional {
			requiredFields = append(requiredFields, f)
		}
	}
	return requiredFields
}

func NewResourceID(expectedFields ...ResourceIDField) *ResourceID {
	return newResourceIDWithSeparators([]string{defaultSeparator}, expectedFields...)
}

// Deprecated: Use NewResourceID instead
// We should standardize on a single separator, so that function should only be used for old resources
// On major versions, switch to NewResourceID and remove uses of this function
func NewResourceIDWithLegacySeparator(legacySeparator string, expectedFields ...ResourceIDField) *ResourceID {
	return newResourceIDWithSeparators([]string{defaultSeparator, legacySeparator}, expectedFields...)
}

func newResourceIDWithSeparators(separators []string, expectedFields ...ResourceIDField) *ResourceID {
	tfID := &ResourceID{
		separators:     separators,
		expectedFields: expectedFields,
	}
	return tfID
}

// Make creates a resource ID from the given parts
// The parts must have the correct number of fields and types
func (id *ResourceID) Make(parts ...any) string {
	// TODO: Manage optional fields correctly
	var expectedFields []ResourceIDField
	for _, f := range id.expectedFields {
		if !f.Optional {
			expectedFields = append(expectedFields, f)
		}
	}

	if len(parts) != len(expectedFields) {
		panic(fmt.Sprintf("expected %d fields, got %d", len(expectedFields), len(parts))) // This is a coding error, so panic is appropriate
	}
	stringParts := make([]string, len(parts))
	for i, part := range parts {
		// Unwrap pointers
		if reflect.ValueOf(part).Kind() == reflect.Ptr {
			part = reflect.ValueOf(part).Elem().Interface()
		}
		expectedField := expectedFields[i]
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
	requiredFields := id.RequiredFields()

	// Try with optional fields
	parts, err := split(resourceID, id.expectedFields, id.separators)
	if err == nil {
		return parts, nil
	}
	if len(requiredFields) == len(id.expectedFields) {
		return nil, err
	}

	// Try without optional fields
	parts, err = split(resourceID, requiredFields, id.separators)
	if err != nil {
		return nil, err
	}
	return parts, nil
}

// Split parses a resource ID into its parts
// The parts will be cast to the expected types
func split(resourceID string, expectedFields []ResourceIDField, separators []string) ([]any, error) {
	for _, sep := range separators {
		parts := strings.Split(resourceID, sep)
		if len(parts) == len(expectedFields) {
			partsAsAny := make([]any, len(parts))
			for i, part := range parts {
				expectedField := expectedFields[i]
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

	expectedFieldNames := make([]string, len(expectedFields))
	for i, f := range expectedFields {
		expectedFieldNames[i] = f.Name
	}
	return nil, fmt.Errorf("id %q does not match expected format. Should be in the format: %s", resourceID, strings.Join(expectedFieldNames, defaultSeparator))
}
