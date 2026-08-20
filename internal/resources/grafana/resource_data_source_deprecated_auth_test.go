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

func TestCheckDeprecatedPrometheusAuth_AssumeRoleArn(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"assumeRoleArn":"arn:aws:iam::123456789012:role/my-role"}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, 1, len(diags))
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "Incorrect key for IAM role assumption", diags[0].Summary)
}

func TestCheckDeprecatedPrometheusAuth_AssumeRoleArnWithCorrectKey(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"sigV4AssumeRoleArn":"arn:aws:iam::123456789012:role/my-role"}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, 0, len(diags))
}

func TestCheckDeprecatedPrometheusAuth_AssumeRoleArnEmpty(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"assumeRoleArn":""}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, 0, len(diags))
}

func TestCheckDeprecatedPrometheusAuth_AssumeRoleArnAmazonPrometheusPlugin(t *testing.T) {
	// assumeRoleArn is the correct key for the plugin type — no warning expected
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-amp",
		"type":              "grafana-amazonprometheus-datasource",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"assumeRoleArn":"arn:aws:iam::123456789012:role/my-role"}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	require.Equal(t, 0, len(diags))
}
