// Package integrationschemas describes Grafana Alerting integrations (the settings of Slack, email,
// webhook and so on) in terms a Terraform resource can use: IntegrationType and Field.
//
// They come from a committed snapshot of Grafana's integration type schemas (see README.md). The
// package has no dependency on the Terraform plugin framework.
package integrationschemas

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed integration-schemas.json
var snapshotJSON []byte

// The UI element kinds of a snapshot field (Field.Element).
const (
	elementInput        = "input"
	elementTextarea     = "textarea"
	elementCheckbox     = "checkbox"
	elementSelect       = "select"
	elementStringArray  = "string_array"
	elementKeyValueMap  = "key_value_map"
	elementSubform      = "subform"
	elementSubformArray = "subform_array"
)

// Source says where a snapshot comes from: a file in a Grafana repository at a commit.
type Source struct {
	Repository string `json:"repository"`
	SHA        string `json:"sha"`
	Path       string `json:"path"`
}

// Snapshot is the decoded snapshot file: the integration types that can be created, reduced to what
// this package uses.
type Snapshot struct {
	Source Source            `json:"source"`
	Types  []IntegrationType `json:"types"`
}

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ParseSnapshot decodes and validates a snapshot. It fails on keys it does not know, and on field kinds
// this package cannot represent, rather than dropping them.
func ParseSnapshot(data []byte) (*Snapshot, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var s Snapshot
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding integration schema snapshot: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Snapshot) validate() error {
	if s.Source.Repository == "" || s.Source.Path == "" || !shaPattern.MatchString(s.Source.SHA) {
		return fmt.Errorf("integration schema snapshot has no valid source: %+v", s.Source)
	}
	for i, t := range s.Types {
		if t.APIType == "" || t.Name == "" || t.Version == "" {
			return fmt.Errorf("integration schema type %d has no type, name or version", i)
		}
		if i > 0 && s.Types[i-1].APIType >= t.APIType {
			return fmt.Errorf("integration schema types are not sorted and unique at %q", t.APIType)
		}
		if err := validateFields(t.APIType+"/"+t.Version, t.Fields); err != nil {
			return err
		}
	}
	return nil
}

func validateFields(path string, fields []*Field) error {
	names := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		p := path + "." + f.APIName
		if f.APIName == "" {
			return fmt.Errorf("integration schema field at %s has no name", path)
		}
		if _, ok := names[f.APIName]; ok {
			return fmt.Errorf("duplicate integration schema field %s", p)
		}
		names[f.APIName] = struct{}{}

		switch f.Element {
		case elementInput, elementTextarea, elementCheckbox, elementStringArray, elementKeyValueMap:
			if len(f.Children) > 0 || len(f.AllowedValues) > 0 {
				return fmt.Errorf("integration schema field %s (%s) unexpectedly has options or subfields", p, f.Element)
			}
		case elementSelect:
			if len(f.AllowedValues) == 0 {
				return fmt.Errorf("integration schema select field %s has no options", p)
			}
			if len(f.Children) > 0 {
				return fmt.Errorf("integration schema select field %s unexpectedly has subfields", p)
			}
		case elementSubform, elementSubformArray:
			if len(f.AllowedValues) > 0 {
				return fmt.Errorf("integration schema field %s (%s) unexpectedly has options", p, f.Element)
			}
			if err := validateFields(p, f.Children); err != nil {
				return err
			}
		default:
			return fmt.Errorf("integration schema field %s has unsupported element %q", p, f.Element)
		}
	}
	return nil
}

// Marshal encodes a snapshot the way it is committed: one line per field, so that a change to a field
// is a one-line diff. Types must be sorted by type.
func (s *Snapshot) Marshal() ([]byte, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("{\n  \"source\": " + compactJSON(s.Source) + ",\n  \"types\": [\n")
	for i, t := range s.Types {
		fmt.Fprintf(&b, "    {\n      \"type\": %s,\n      \"name\": %s,\n      \"version\": %s,\n      \"fields\": [\n",
			compactJSON(t.APIType), compactJSON(t.Name), compactJSON(t.Version))
		writeFields(&b, t.Fields, "        ")
		b.WriteString("      ]\n    }" + comma(i, len(s.Types)) + "\n")
	}
	b.WriteString("  ]\n}\n")
	return b.Bytes(), nil
}

// writeFields writes one line per field. A field with subfields is opened on its own line, followed by
// its subfields.
func writeFields(b *bytes.Buffer, fields []*Field, indent string) {
	for i, f := range fields {
		last := comma(i, len(fields))
		if len(f.Children) == 0 {
			b.WriteString(indent + compactJSON(f) + last + "\n")
			continue
		}
		open := *f
		open.Children = nil
		b.WriteString(indent + strings.TrimSuffix(compactJSON(open), "}") + `,"fields":[` + "\n")
		writeFields(b, f.Children, indent+"  ")
		b.WriteString(indent + "]}" + last + "\n")
	}
}

func comma(i, n int) string {
	if i < n-1 {
		return ","
	}
	return ""
}

// compactJSON encodes v on one line, without escaping HTML characters.
func compactJSON(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err) // the snapshot types always encode
	}
	return strings.TrimSuffix(b.String(), "\n")
}
