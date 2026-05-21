package appplatform

import (
	"context"
	"testing"

	provisioningv0alpha1 "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/require"
)

func TestRepositorySchemaIncludesWebhookBlock(t *testing.T) {
	var resp tfresource.SchemaResponse
	Repository().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &resp)

	require.False(t, resp.Diagnostics.HasError())

	specBlock, ok := resp.Schema.Blocks["spec"].(schema.SingleNestedBlock)
	require.True(t, ok)

	webhookBlock, ok := specBlock.Blocks["webhook"].(schema.SingleNestedBlock)
	require.True(t, ok)

	baseURLAttr, ok := webhookBlock.Attributes["base_url"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, baseURLAttr.Optional)
}

func TestParseRepositorySpecIncludesWebhookBaseURL(t *testing.T) {
	ctx := context.Background()

	src := types.ObjectValueMust(repositorySpecType.AttrTypes, map[string]attr.Value{
		"title":       types.StringValue("Git Sync repository"),
		"description": types.StringNull(),
		"workflows":   types.ListNull(types.StringType),
		"sync": types.ObjectValueMust(repositorySyncType.AttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(false),
			"target":           types.StringValue(string(provisioningv0alpha1.SyncTargetTypeInstance)),
			"interval_seconds": types.Int64Null(),
		}),
		"type": types.StringValue(string(provisioningv0alpha1.GitHubRepositoryType)),
		"github": types.ObjectValueMust(repositoryGitHubType.AttrTypes, map[string]attr.Value{
			"url":                         types.StringValue("https://github.com/grafana/terraform-provider-grafana"),
			"branch":                      types.StringValue("main"),
			"path":                        types.StringValue("examples"),
			"generate_dashboard_previews": types.BoolValue(false),
		}),
		"git":        types.ObjectNull(repositoryGitType.AttrTypes),
		"bitbucket":  types.ObjectNull(repositoryBitbucketType.AttrTypes),
		"gitlab":     types.ObjectNull(repositoryGitLabType.AttrTypes),
		"local":      types.ObjectNull(repositoryLocalType.AttrTypes),
		"connection": types.ObjectNull(repositoryConnectionType.AttrTypes),
		"webhook": types.ObjectValueMust(repositoryWebhookType.AttrTypes, map[string]attr.Value{
			"base_url": types.StringValue("https://hooks.example.com"),
		}),
	})

	dst := &ProvisioningRepository{}
	diags := parseRepositorySpec(ctx, src, dst)

	require.False(t, diags.HasError())
	require.NotNil(t, dst.Spec.Webhook)
	require.Equal(t, "https://hooks.example.com", dst.Spec.Webhook.BaseURL)
}

func TestSaveRepositorySpecIncludesWebhookBaseURL(t *testing.T) {
	ctx := context.Background()

	src := &ProvisioningRepository{
		Spec: provisioningv0alpha1.RepositorySpec{
			Title: "Git Sync repository",
			Sync: provisioningv0alpha1.SyncOptions{
				Enabled: false,
				Target:  provisioningv0alpha1.SyncTargetTypeInstance,
			},
			Type: provisioningv0alpha1.GitHubRepositoryType,
			GitHub: &provisioningv0alpha1.GitHubRepositoryConfig{
				URL:    "https://github.com/grafana/terraform-provider-grafana",
				Branch: "main",
				Path:   "examples",
			},
			Webhook: &provisioningv0alpha1.WebhookConfig{
				BaseURL: "https://hooks.example.com",
			},
		},
	}

	dst := &ResourceModel{}
	diags := saveRepositorySpec(ctx, src, dst)

	require.False(t, diags.HasError())

	var spec RepositorySpecModel
	diags = dst.Spec.As(ctx, &spec, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())

	var webhook RepositoryWebhookModel
	diags = spec.Webhook.As(ctx, &webhook, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "https://hooks.example.com", webhook.BaseURL.ValueString())
}
