package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestCheckDeprecatedPrometheusAuth_SigV4(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"sigV4Auth":true}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 1 {
		t.Errorf("Expected 1 diagnostic, got %d", len(diags))
	}

	require.Equal(t, len(diags), 1)
	require.Equal(t, diags[0].Severity, diag.Warning)
}

func TestCheckDeprecatedPrometheusAuth_Azure(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"azureCredentials":{}}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 1 {
		t.Errorf("Expected 1 diagnostic, got %d", len(diags))
	}

	require.Equal(t, len(diags), 1)
	require.Equal(t, diags[0].Severity, diag.Warning)
}

func TestCheckDeprecatedPrometheusAuth_NoDeprecatedAuth(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name": "test-prometheus",
		"type": "prometheus",
		"url":  "http://localhost:9090",
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, len(diags), 0)
}

func TestCheckDeprecatedPrometheusAuth_NonPrometheusDataSource(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-loki",
		"type":              "loki",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"sigV4Auth":true}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, len(diags), 0)
}
