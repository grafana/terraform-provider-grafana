package fleetmanagement

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipelineContentsValidator_Description(t *testing.T) {
	ctx := context.Background()
	validator := &pipelineContentsValidator{}

	desc := validator.Description(ctx)
	require.Contains(t, desc, "ALLOY")
	require.Contains(t, desc, "OTEL")
}

func TestPipelineContentsValidator_MarkdownDescription(t *testing.T) {
	ctx := context.Background()
	validator := &pipelineContentsValidator{}

	desc := validator.MarkdownDescription(ctx)
	require.Contains(t, desc, "ALLOY")
	require.Contains(t, desc, "OTEL")
}

// TestPipelineContentsValidator_ValidateContents tests the validation logic
// that was previously in AlloyConfigValue.ValidateAttribute.
// This tests the same behavior but at the resource level, allowing
// validation to be conditional based on config_type.
func TestPipelineContentsValidator_ValidateContents(t *testing.T) {
	t.Run("valid Alloy config with ALLOY type", func(t *testing.T) {
		diags := validatePipelineContents("logging {}", "ALLOY")
		require.False(t, diags.HasError())
	})

	t.Run("valid Alloy config with empty type defaults to ALLOY", func(t *testing.T) {
		diags := validatePipelineContents("logging {}", "")
		require.False(t, diags.HasError())
	})

	t.Run("invalid Alloy config with ALLOY type", func(t *testing.T) {
		diags := validatePipelineContents("invalid alloy config", "ALLOY")
		require.True(t, diags.HasError())
		require.Equal(t, 1, diags.ErrorsCount())
		require.Contains(t, diags.Errors()[0].Summary(), "Invalid Alloy configuration")
	})

	t.Run("valid YAML config with OTEL type", func(t *testing.T) {
		yamlConfig := "receivers:\n  otlp:\n    protocols:\n      grpc:"
		diags := validatePipelineContents(yamlConfig, "OTEL")
		require.False(t, diags.HasError())
	})

	t.Run("invalid YAML config with OTEL type", func(t *testing.T) {
		diags := validatePipelineContents(":\ninvalid", "OTEL")
		require.True(t, diags.HasError())
		require.Equal(t, 1, diags.ErrorsCount())
		require.Contains(t, diags.Errors()[0].Summary(), "Invalid OTEL configuration")
	})

	t.Run("Alloy config with comment is valid", func(t *testing.T) {
		diags := validatePipelineContents("// this is a valid Alloy comment", "ALLOY")
		require.False(t, diags.HasError())
	})

	t.Run("complex Alloy config is valid", func(t *testing.T) {
		alloyConfig := `// Export Alloy metrics in memory.
prometheus.exporter.self "integrations_alloy_health" { }

discovery.relabel "integrations_alloy_health" {
	targets = prometheus.exporter.self.integrations_alloy_health.targets

	rule {
		action       = "replace"
		target_label = "collector_id"
		replacement  = argument.attributes.value["collector.ID"]
	}

	rule {
		target_label = "instance"
		replacement  = constants.hostname
	}

	rule {
		target_label = "job"
		replacement  = "integrations/alloy"
	}
}

prometheus.scrape "integrations_alloy_health" {
	targets = array.concat(
		discovery.relabel.integrations_alloy_health.output,
	)
	forward_to = [prometheus.relabel.integrations_alloy_health.receiver]
	job_name   = "integrations/alloy"
}`
		diags := validatePipelineContents(alloyConfig, "ALLOY")
		require.False(t, diags.HasError())
	})

	t.Run("complex OTEL config is valid", func(t *testing.T) {
		otelConfig := `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`
		diags := validatePipelineContents(otelConfig, "OTEL")
		require.False(t, diags.HasError())
	})
}
