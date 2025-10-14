package common

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type ResourceIDFieldType string

const (
	ResourceIDSeparator       = ":"
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
	expectedFields []ResourceIDField
}

func (id *ResourceID) Fields() []ResourceIDField {
	return id.expectedFields
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
	return &ResourceID{
		expectedFields: expectedFields,
	}
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

	return strings.Join(stringParts, ResourceIDSeparator)
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
	parts, err := split(resourceID, id.expectedFields)
	if err == nil {
		return parts, nil
	}
	if len(requiredFields) == len(id.expectedFields) {
		return nil, err
	}

	// Try without optional fields
	parts, err = split(resourceID, requiredFields)
	if err != nil {
		return nil, err
	}
	return parts, nil
}

// Split parses a resource ID into its parts
// The parts will be cast to the expected types
func split(resourceID string, expectedFields []ResourceIDField) ([]any, error) {
	parts := strings.Split(resourceID, ResourceIDSeparator)
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

	expectedFieldNames := make([]string, len(expectedFields))
	for i, f := range expectedFields {
		expectedFieldNames[i] = f.Name
	}
	return nil, fmt.Errorf("id %q does not match expected format. Should be in the format: %s", resourceID, strings.Join(expectedFieldNames, ResourceIDSeparator))
}
