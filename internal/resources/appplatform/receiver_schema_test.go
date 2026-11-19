package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
