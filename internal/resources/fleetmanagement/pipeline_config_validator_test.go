package fleetmanagement

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineContentsValidator_Description(t *testing.T) {
	ctx := context.Background()
	validator := &pipelineContentsValidator{}

	desc := validator.Description(ctx)
	assert.Contains(t, desc, "ALLOY")
	assert.Contains(t, desc, "OTEL")
}

func TestPipelineContentsValidator_MarkdownDescription(t *testing.T) {
	ctx := context.Background()
	validator := &pipelineContentsValidator{}

	desc := validator.MarkdownDescription(ctx)
	assert.Contains(t, desc, "ALLOY")
	assert.Contains(t, desc, "OTEL")
}

// TestPipelineContentsValidator_ValidateContents tests the validation logic
// that was previously in AlloyConfigValue.ValidateAttribute.
// This tests the same behavior but at the resource level, allowing
// validation to be conditional based on config_type.
func TestPipelineContentsValidator_ValidateContents(t *testing.T) {
	t.Run("valid Alloy config with ALLOY type", func(t *testing.T) {
		diags := validatePipelineContents("logging {}", "ALLOY")
		assert.False(t, diags.HasError())
	})

	t.Run("valid Alloy config with empty type defaults to ALLOY", func(t *testing.T) {
		diags := validatePipelineContents("// valid comment", "")
		assert.False(t, diags.HasError())
	})

	t.Run("invalid Alloy config with ALLOY type", func(t *testing.T) {
		diags := validatePipelineContents("invalid alloy config", "ALLOY")
		assert.True(t, diags.HasError())
		assert.Equal(t, 1, diags.ErrorsCount())
		assert.Contains(t, diags.Errors()[0].Summary(), "Invalid Alloy configuration")
	})

	t.Run("valid YAML config with OTEL type", func(t *testing.T) {
		yamlConfig := "receivers:\n  otlp:\n    protocols:\n      grpc:"
		diags := validatePipelineContents(yamlConfig, "OTEL")
		assert.False(t, diags.HasError())
	})

	t.Run("invalid YAML config with OTEL type", func(t *testing.T) {
		diags := validatePipelineContents(":\ninvalid", "OTEL")
		assert.True(t, diags.HasError())
		assert.Equal(t, 1, diags.ErrorsCount())
		assert.Contains(t, diags.Errors()[0].Summary(), "Invalid OTEL configuration")
	})

	t.Run("Alloy config with comment is valid", func(t *testing.T) {
		diags := validatePipelineContents("// this is a valid Alloy comment", "ALLOY")
		assert.False(t, diags.HasError())
	})

	t.Run("complex Alloy config is valid", func(t *testing.T) {
		alloyConfig := `prometheus.exporter.self "alloy" { }`
		diags := validatePipelineContents(alloyConfig, "ALLOY")
		assert.False(t, diags.HasError())
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
		assert.False(t, diags.HasError())
	})
}

func TestConfigTypeValidation(t *testing.T) {
	// Test that YAML config is not valid River
	t.Run("OTEL config is valid YAML", func(t *testing.T) {
		otelConfig := `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`
		_, err := parseRiver(otelConfig)
		assert.Error(t, err)
	})

	// Test that River config is not valid YAML
	t.Run("River config is not valid YAML", func(t *testing.T) {
		alloyConfig := `
prometheus.exporter.self "alloy" { }

prometheus.scrape "self" {
    targets    = prometheus.exporter.self.alloy.targets
    forward_to = [prometheus.remote_write.default.receiver]
}
`
		_, err := parseYAML(alloyConfig)
		assert.Error(t, err)
	})
}
