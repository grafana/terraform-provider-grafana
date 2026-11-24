package main

import (
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/stretchr/testify/require"
)

var testSource = integrationschemas.Source{Repository: "grafana/grafana", SHA: strings.Repeat("a", 40), Path: "list.json"}

const (
	upstreamHeader = `"apiVersion": "notifications.alerting.grafana.app/v1beta1", "kind": "IntegrationTypeSchemaList"`
	itemHead       = `"apiVersion": "notifications.alerting.grafana.app/v1beta1", "kind": "IntegrationTypeSchema", "metadata": {"name": "x", "namespace": "default"}`
)

// upstreamItems wraps integration type schemas in a list response.
func testList(items ...string) []byte {
	for i, it := range items {
		items[i] = `{` + itemHead + `, "spec": ` + it + `}`
	}
	return []byte(`{` + upstreamHeader + `, "items": [` + strings.Join(items, ",") + `]}`)
}

func testType(typ string, versions ...string) string {
	return `{"type": "` + typ + `", "currentVersion": "v1", "name": "` + strings.ToUpper(typ[:1]) + typ[1:] + `", "versions": [` + strings.Join(versions, ",") + `]}`
}

func testVersion(version string, canCreate bool, fields ...string) string {
	create := "false"
	if canCreate {
		create = "true"
	}
	return `{"version": "` + version + `", "canCreate": ` + create + `, "options": [` + strings.Join(fields, ",") + `]}`
}

func TestReduce(t *testing.T) {
	data := testList(
		// Out of order: Reduce sorts by type.
		testType("slack",
			testVersion("v0mimir1", false, `{"element": "input", "propertyName": "legacy"}`),
			testVersion("v1", true,
				`{"element": "input", "propertyName": "url", "label": "URL", "description": "", "required": true, "secure": true, "placeholder": "p", "validationRule": "r", "inputType": "text"}`,
				`{"element": "input", "propertyName": "token", "label": "Token", "description": "The token", "required": true, "dependsOn": "url"}`,
				`{"element": "input", "propertyName": "channel", "label": "Channel", "required": true, "showWhen": {"field": "url", "is": "x"}}`,
				`{"element": "select", "propertyName": "mode", "label": "Mode", "selectOptions": [{"label": "A", "value": "a"}, {"label": "B", "value": "b", "description": "d"}]}`,
				`{"element": "subform", "propertyName": "sub", "label": "Sub", "subformOptions": [{"element": "checkbox", "propertyName": "on", "label": "On", "required": true}]}`,
			),
		),
		// Only versions that cannot be created: left out.
		testType("line", testVersion("v1", false, `{"element": "input", "propertyName": "a"}`)),
		testType("email", testVersion("v1", true, `{"element": "input", "propertyName": "addresses", "label": "Addresses"}`)),
	)

	out, err := reduce(data, testSource)
	require.NoError(t, err)

	s, err := integrationschemas.ParseSnapshot(out)
	require.NoError(t, err)
	require.Equal(t, testSource, s.Source)
	require.Len(t, s.Types, 2)
	require.Equal(t, "email", s.Types[0].APIType)
	require.Equal(t, "Email", s.Types[0].Name)

	slack := s.Types[1]
	require.Equal(t, "slack", slack.APIType)
	require.Equal(t, "v1", slack.Version, "versions that cannot be created are left out")
	require.Equal(t, []*integrationschemas.Field{
		{APIName: "url", Element: "input", Description: "URL", Required: true, Secure: true},
		{APIName: "token", Element: "input", Description: "The token"}, // depends on another field: not required
		{APIName: "channel", Element: "input", Description: "Channel"}, // only shown with another field: not required
		{APIName: "mode", Element: "select", Description: "Mode", AllowedValues: []string{"a", "b"}},
		{APIName: "sub", Element: "subform", Description: "Sub", Children: []*integrationschemas.Field{{APIName: "on", Element: "checkbox", Description: "On", Required: true}}},
	}, slack.Fields)

	again, err := reduce(data, testSource)
	require.NoError(t, err)
	require.Equal(t, string(out), string(again), "the output is deterministic")
}

func TestReduceRejectsUnsupportedResponses(t *testing.T) {
	field := func(f string) []byte {
		return testList(testType("a", testVersion("v1", true, f)))
	}

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"unexpected API version", []byte(`{"apiVersion": "notifications.alerting.grafana.app/v2", "kind": "IntegrationTypeSchemaList", "items": []}`), "unsupported integration type schemas"},
		{"unexpected item kind", []byte(`{` + upstreamHeader + `, "items": [{"apiVersion": "notifications.alerting.grafana.app/v1beta1", "kind": "Other", "metadata": {"name": "x"}, "spec": {"type": "a", "currentVersion": "v1", "name": "A", "versions": []}}]}`), "unsupported integration type schema item"},
		{"unknown key", []byte(`{` + upstreamHeader + `, "items": [], "extra": true}`), "unknown field"},
		{"unknown field key", field(`{"element": "input", "propertyName": "a", "surprise": 1}`), "unknown field"},
		{"unsupported element", field(`{"element": "dial", "propertyName": "a"}`), `unsupported element "dial"`},
		{"numeric select value", field(`{"element": "select", "propertyName": "a", "selectOptions": [{"label": "One", "value": 1}]}`), "non-string value"},
		{"more than one version that can be created", testList(testType("a", testVersion("v1", true), testVersion("v2", true))), "more than one version"},
		{"duplicate field", testList(testType("a", testVersion("v1", true, `{"element": "input", "propertyName": "a"}`, `{"element": "input", "propertyName": "a"}`))), "duplicate integration schema field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reduce(tt.data, testSource)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestReduceIgnoresProblemsInVersionsThatCannotBeCreated(t *testing.T) {
	out, err := reduce(testList(testType("a",
		testVersion("v0", false, `{"element": "dial", "propertyName": "a"}`),
		testVersion("v1", true, `{"element": "input", "propertyName": "a"}`),
	)), testSource)
	require.NoError(t, err)
	_, err = integrationschemas.ParseSnapshot(out)
	require.NoError(t, err)
}
