package integrationschemas

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
)

var (
	loadOnce sync.Once
	loaded   []IntegrationType
	loadErr  error
)

// Load returns the integration types that can be created, built from the embedded snapshot and
// sorted by type, with the Terraform-specific parts filled in. The result is cached and must not be
// modified.
func Load() ([]IntegrationType, error) {
	loadOnce.Do(func() {
		snap, err := ParseSnapshot(snapshotJSON)
		if err != nil {
			loadErr = err
			return
		}
		loaded, loadErr = build(snap)
	})
	return loaded, loadErr
}

// build fills in the Terraform-specific parts of the snapshot's integration types, in place.
func build(s *Snapshot) ([]IntegrationType, error) {
	names := map[string]string{}
	usedOverrides := map[string]struct{}{}
	usedTypeOverrides := map[string]struct{}{}
	for i := range s.Types {
		it := &s.Types[i]

		base := tfNameOf(it.APIType)
		if name, ok := typeNameOverrides[it.APIType]; ok {
			base = name
			usedTypeOverrides[it.APIType] = struct{}{}
		}
		// The version is part of the name, so that every version has a schema of its own and adding a
		// version later does not change what an existing name means.
		it.TFName = base + "_" + it.Version
		if other, ok := names[it.TFName]; ok {
			return nil, fmt.Errorf("integration types %q and %q both map to the name %q", other, it.APIType, it.TFName)
		}
		names[it.TFName] = it.APIType

		if err := buildFields(it.APIType, it.Version, nil, it.Fields, usedOverrides); err != nil {
			return nil, err
		}
	}
	for typ := range typeNameOverrides {
		if _, ok := usedTypeOverrides[typ]; !ok {
			return nil, fmt.Errorf("type name override %q does not match an integration type in the schema snapshot", typ)
		}
	}
	for key := range fieldOverrides {
		if _, ok := usedOverrides[key]; !ok {
			return nil, fmt.Errorf("field override %q does not match a field in the integration schema snapshot", key)
		}
	}
	return s.Types, nil
}

func buildFields(integrationType, version string, parent []string, fields []*Field, used map[string]struct{}) error {
	names := map[string]string{}
	for _, f := range fields {
		f.Path = append(append([]string{}, parent...), f.APIName)
		f.TFName = tfNameOf(f.APIName)
		overrideKey := integrationType + "/" + version + "/" + strings.Join(f.Path, ".")
		override, hasOverride := fieldOverrides[overrideKey]
		if hasOverride {
			used[overrideKey] = struct{}{}
			if override.tfName != "" {
				f.TFName = override.tfName
			}
		}
		if other, ok := names[f.TFName]; ok {
			return fmt.Errorf("fields %q and %q of %s/%s both map to %q", other, f.APIName, integrationType, version, f.TFName)
		}
		names[f.TFName] = f.APIName

		f.Kind = kindOf(f.Element, override)
		f.Sensitive = f.Secure || override.sensitive
		f.Required = f.Required && !override.optional
		if override.appendDescription != "" {
			f.Description += " " + override.appendDescription
		}
		if override.freeform {
			if len(f.AllowedValues) > 0 {
				f.Description += " Usual values are `" + strings.Join(f.AllowedValues, "`, `") + "`."
			}
			f.AllowedValues = nil
		}
		if err := buildFields(integrationType, version, f.Path, f.Children, used); err != nil {
			return err
		}
	}
	return nil
}

// kindOf returns how a snapshot field is represented in Terraform: the override if there is one,
// else what its UI element implies.
func kindOf(el string, override fieldOverride) FieldKind {
	switch override.value {
	case overrideInt:
		return KindInt
	case overrideJoinedStringList:
		return KindJoinedStringList
	}
	switch el {
	case elementCheckbox:
		return KindBool
	case elementStringArray:
		return KindStringList
	case elementKeyValueMap:
		return KindStringMap
	case elementSubform:
		return KindObject
	case elementSubformArray:
		return KindObjectList
	default: // input, textarea, select
		return KindString
	}
}

// tfNameOf converts an API name to a Terraform name: camelCase and kebab-case become snake_case,
// and existing snake_case is kept.
func tfNameOf(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		switch {
		case r == '-':
			b.WriteRune('_')
		case unicode.IsUpper(r):
			if i > 0 && runes[i-1] != '_' && runes[i-1] != '-' {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
