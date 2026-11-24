package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestReceiverIntegrationBlocksAreValidFrameworkSchema(t *testing.T) {
	integrationTypes, err := integrationschemas.Load()
	require.NoError(t, err)
	require.NoError(t, validateReceiverIntegrationNames(integrationTypes))

	blocks := receiverIntegrationBlocks(integrationTypes)
	require.Len(t, blocks, len(integrationTypes))

	// ValidateImplementation checks names, defaults and required/optional/computed combinations
	// for every nested attribute and block.
	s := schema.Schema{Blocks: map[string]schema.Block{
		"spec": schema.SingleNestedBlock{Blocks: blocks},
	}}
	diags := s.ValidateImplementation(context.Background())
	require.False(t, diags.HasError(), "%v", diags)
}

func TestReceiverCaseInsensitiveSelects(t *testing.T) {
	integrationTypes, err := integrationschemas.Load()
	require.NoError(t, err)
	blocks := receiverIntegrationBlocks(integrationTypes)

	attrs := blocks["telegram_v1"].(schema.ListNestedBlock).NestedObject.Attributes
	validators := attrs["parse_mode"].(schema.StringAttribute).Validators
	require.Len(t, validators, 1)

	check := func(value string) bool {
		resp := &validator.StringResponse{}
		validators[0].ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
		return !resp.Diagnostics.HasError()
	}
	for _, ok := range []string{"HTML", "html", "markdownv2", "None"} {
		require.True(t, check(ok), ok)
	}
	require.False(t, check("HTLM"))

	// Other selects stay case-sensitive.
	kafka := blocks["kafka_v1"].(schema.ListNestedBlock).NestedObject.Attributes["api_version"].(schema.StringAttribute).Validators
	resp := &validator.StringResponse{}
	kafka[0].ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue("V2")}, resp)
	require.True(t, resp.Diagnostics.HasError())
}
