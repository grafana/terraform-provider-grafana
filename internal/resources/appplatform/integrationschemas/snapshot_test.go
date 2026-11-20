package integrationschemas

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
	s, err := ParseSnapshot(snapshotJSON)
	require.NoError(t, err)
	require.NotEmpty(t, s.Types)
	require.Equal(t, "grafana/grafana", s.Source.Repository)

	var slack IntegrationType
	for _, it := range s.Types {
		if it.APIType == "slack" {
			slack = it
		}
	}
	require.Equal(t, "slack", slack.APIType)
	require.Equal(t, "v1", slack.Version)
	require.NotEmpty(t, slack.Fields)
}

// TestSnapshotIsCanonical makes sure that the committed file is what Reduce writes, so a hand edit or a
// change of the format shows up here instead of as a diff the next time the file is generated.
func TestSnapshotIsCanonical(t *testing.T) {
	s, err := ParseSnapshot(snapshotJSON)
	require.NoError(t, err)
	out, err := s.Marshal()
	require.NoError(t, err)
	require.Equal(t, string(snapshotJSON), string(out), "integration-schemas.json is not what `make update-integration-schemas` writes; do not edit it by hand")
}

func TestParseRejectsUnsupportedSnapshots(t *testing.T) {
	const source = `{"repository": "grafana/grafana", "sha": "9cd436783d30dbef3adbc94cc072c3e5f50531b1", "path": "list.json"}`
	snap := func(types string) []byte {
		return []byte(`{"source": ` + source + `, "types": [` + types + `]}`)
	}
	typ := func(name, fields string) string {
		return `{"type": "` + name + `", "name": "X", "version": "v1", "fields": [` + fields + `]}`
	}

	_, err := ParseSnapshot(snap(typ("a", `{"name": "a", "element": "input"}`)))
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"unknown key", []byte(`{"source": ` + source + `, "types": [], "extra": true}`), "unknown field"},
		{"no source", []byte(`{"types": []}`), "no valid source"},
		{"source without a full sha", []byte(`{"source": {"repository": "r", "sha": "9cd436", "path": "p"}, "types": []}`), "no valid source"},
		{"unsorted types", snap(typ("b", `{"name": "a", "element": "input"}`) + "," + typ("a", `{"name": "a", "element": "input"}`)), "not sorted"},
		{"no version", snap(`{"type": "a", "name": "X", "fields": []}`), "no type, name or version"},
		{"unsupported element", snap(typ("a", `{"name": "a", "element": "dial"}`)), `unsupported element "dial"`},
		{"duplicate field", snap(typ("a", `{"name": "a", "element": "input"}, {"name": "a", "element": "input"}`)), "duplicate integration schema field"},
		{"select without options", snap(typ("a", `{"name": "a", "element": "select"}`)), "has no options"},
		{"input with options", snap(typ("a", `{"name": "a", "element": "input", "options": ["x"]}`)), "unexpectedly has options"},
		{"checkbox with subfields", snap(typ("a", `{"name": "a", "element": "checkbox", "fields": [{"name": "b", "element": "input"}]}`)), "unexpectedly has options or subfields"},
		{"null field", snap(typ("a", `null`)), "is null"},
		{"null subfield", snap(typ("a", `{"name": "a", "element": "subform", "fields": [null]}`)), "is null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSnapshot(tt.data)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestMarshalSnapshotWritesOneLinePerField(t *testing.T) {
	s := &Snapshot{
		Source: Source{Repository: "grafana/grafana", SHA: strings.Repeat("a", 40), Path: "list.json"},
		Types: []IntegrationType{{APIType: "a", Name: "A", Version: "v1", Fields: []*Field{
			{APIName: "x", Element: elementInput, Description: "a <b> & c"},
			{APIName: "sub", Element: elementSubform, Children: []*Field{{APIName: "y", Element: elementCheckbox, Required: true}}},
			{APIName: "z", Element: elementSelect, AllowedValues: []string{"one", "two"}, Secure: true},
		}}},
	}
	out, err := s.Marshal()
	require.NoError(t, err)
	require.Contains(t, string(out), `{"name":"x","element":"input","description":"a <b> & c"},`)
	require.Contains(t, string(out), `{"name":"sub","element":"subform","fields":[`+"\n")
	require.Contains(t, string(out), `{"name":"y","element":"checkbox","required":true}`)
	require.Contains(t, string(out), `{"name":"z","element":"select","secure":true,"options":["one","two"]}`)

	parsed, err := ParseSnapshot(out)
	require.NoError(t, err)
	require.Equal(t, s, parsed)
	require.True(t, json.Valid(out))
}
