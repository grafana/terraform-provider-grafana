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

func TestRepositorySchemaIncludesNewBlocks(t *testing.T) {
	var resp tfresource.SchemaResponse
	Repository().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &resp)

	require.False(t, resp.Diagnostics.HasError())

	specBlock, ok := resp.Schema.Blocks["spec"].(schema.SingleNestedBlock)
	require.True(t, ok)

	ghe, ok := specBlock.Blocks["github_enterprise"].(schema.SingleNestedBlock)
	require.True(t, ok)
	serverURL, ok := ghe.Attributes["server_url"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, serverURL.Optional)

	branch, ok := specBlock.Blocks["branch"].(schema.SingleNestedBlock)
	require.True(t, ok)
	_, ok = branch.Attributes["name_template"].(schema.StringAttribute)
	require.True(t, ok)

	pullRequest, ok := specBlock.Blocks["pull_request"].(schema.SingleNestedBlock)
	require.True(t, ok)
	_, ok = pullRequest.Attributes["title_template"].(schema.StringAttribute)
	require.True(t, ok)

	commit, ok := specBlock.Blocks["commit"].(schema.SingleNestedBlock)
	require.True(t, ok)
	_, ok = commit.Attributes["signing_method"].(schema.StringAttribute)
	require.True(t, ok)
	_, ok = commit.Attributes["signer_name"].(schema.StringAttribute)
	require.True(t, ok)
}

func TestParseRepositorySpecIncludesNewSections(t *testing.T) {
	ctx := context.Background()

	src := types.ObjectValueMust(repositorySpecType.AttrTypes, map[string]attr.Value{
		"title":       types.StringValue("Git Sync repository"),
		"description": types.StringNull(),
		"workflows":   types.ListNull(types.StringType),
		"sync": types.ObjectValueMust(repositorySyncType.AttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(true),
			"target":           types.StringValue(string(provisioningv0alpha1.SyncTargetTypeFolder)),
			"interval_seconds": types.Int64Null(),
		}),
		"type":   types.StringValue(string(provisioningv0alpha1.GitHubEnterpriseRepositoryType)),
		"github": types.ObjectNull(repositoryGitHubType.AttrTypes),
		"github_enterprise": types.ObjectValueMust(repositoryGitHubEnterpriseType.AttrTypes, map[string]attr.Value{
			"server_url":                  types.StringValue("https://ghes.example.com"),
			"url":                         types.StringValue("https://ghes.example.com/example/test"),
			"branch":                      types.StringValue("main"),
			"path":                        types.StringValue("grafana"),
			"generate_dashboard_previews": types.BoolValue(true),
		}),
		"git":        types.ObjectNull(repositoryGitType.AttrTypes),
		"bitbucket":  types.ObjectNull(repositoryBitbucketType.AttrTypes),
		"gitlab":     types.ObjectNull(repositoryGitLabType.AttrTypes),
		"local":      types.ObjectNull(repositoryLocalType.AttrTypes),
		"connection": types.ObjectNull(repositoryConnectionType.AttrTypes),
		"webhook":    types.ObjectNull(repositoryWebhookType.AttrTypes),
		"branch": types.ObjectValueMust(repositoryBranchType.AttrTypes, map[string]attr.Value{
			"name_template":    types.StringValue("grafana/{{title}}-{{random}}"),
			"enforce_template": types.BoolValue(true),
		}),
		"pull_request": types.ObjectValueMust(repositoryPullRequestType.AttrTypes, map[string]attr.Value{
			"title_template":   types.StringValue("Update {{title}}"),
			"enforce_template": types.BoolValue(false),
		}),
		"commit": types.ObjectValueMust(repositoryCommitType.AttrTypes, map[string]attr.Value{
			"single_resource_message_template": types.StringValue("Save {{resourceKind}}: {{title}}"),
			"enforce_template":                 types.BoolValue(true),
			"signer_name":                      types.StringValue("Grafana Bot"),
			"signer_email":                     types.StringValue("bot@example.com"),
			"signing_method":                   types.StringValue(string(provisioningv0alpha1.GPGSigningMethod)),
			"smime_certificate":                types.StringNull(),
		}),
	})

	dst := &ProvisioningRepository{}
	diags := parseRepositorySpec(ctx, src, dst)

	require.False(t, diags.HasError())

	require.NotNil(t, dst.Spec.Commit)
	require.Equal(t, "Save {{resourceKind}}: {{title}}", dst.Spec.Commit.SingleResourceMessageTemplate)
	require.True(t, dst.Spec.Commit.EnforceTemplate)
	require.Equal(t, "Grafana Bot", dst.Spec.Commit.SignerName)
	require.Equal(t, "bot@example.com", dst.Spec.Commit.SignerEmail)
	require.Equal(t, provisioningv0alpha1.GPGSigningMethod, dst.Spec.Commit.SigningMethod)

	require.NotNil(t, dst.Spec.GitHubEnterprise)
	require.Equal(t, "https://ghes.example.com", dst.Spec.GitHubEnterprise.ServerURL)
	require.Equal(t, "https://ghes.example.com/example/test", dst.Spec.GitHubEnterprise.URL)
	require.Equal(t, "main", dst.Spec.GitHubEnterprise.Branch)
	require.Equal(t, "grafana", dst.Spec.GitHubEnterprise.Path)
	require.True(t, dst.Spec.GitHubEnterprise.GenerateDashboardPreviews)

	require.NotNil(t, dst.Spec.Branch)
	require.Equal(t, "grafana/{{title}}-{{random}}", dst.Spec.Branch.NameTemplate)
	require.True(t, dst.Spec.Branch.EnforceTemplate)

	require.NotNil(t, dst.Spec.PullRequest)
	require.Equal(t, "Update {{title}}", dst.Spec.PullRequest.TitleTemplate)
	require.False(t, dst.Spec.PullRequest.EnforceTemplate)
}

func TestSaveRepositorySpecIncludesNewSections(t *testing.T) {
	ctx := context.Background()

	src := &ProvisioningRepository{
		Spec: provisioningv0alpha1.RepositorySpec{
			Title: "Git Sync repository",
			Sync: provisioningv0alpha1.SyncOptions{
				Enabled: true,
				Target:  provisioningv0alpha1.SyncTargetTypeFolder,
			},
			Type: provisioningv0alpha1.GitHubEnterpriseRepositoryType,
			GitHubEnterprise: &provisioningv0alpha1.GitHubEnterpriseRepositoryConfig{
				ServerURL:                 "https://ghes.example.com",
				URL:                       "https://ghes.example.com/example/test",
				Branch:                    "main",
				Path:                      "grafana",
				GenerateDashboardPreviews: true,
			},
			Branch: &provisioningv0alpha1.BranchOptions{
				NameTemplate:    "grafana/{{title}}-{{random}}",
				EnforceTemplate: true,
			},
			PullRequest: &provisioningv0alpha1.PullRequestOptions{
				TitleTemplate:   "Update {{title}}",
				EnforceTemplate: false,
			},
			Commit: &provisioningv0alpha1.CommitOptions{
				SingleResourceMessageTemplate: "Save {{resourceKind}}: {{title}}",
				EnforceTemplate:               true,
				SignerName:                    "Grafana Bot",
				SignerEmail:                   "bot@example.com",
				SigningMethod:                 provisioningv0alpha1.GPGSigningMethod,
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

	var ghe RepositoryGitHubEnterpriseModel
	diags = spec.GitHubEnterprise.As(ctx, &ghe, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "https://ghes.example.com", ghe.ServerURL.ValueString())
	require.Equal(t, "main", ghe.Branch.ValueString())
	require.True(t, ghe.GenerateDashboardPreviews.ValueBool())

	var branch RepositoryBranchModel
	diags = spec.Branch.As(ctx, &branch, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "grafana/{{title}}-{{random}}", branch.NameTemplate.ValueString())
	require.True(t, branch.EnforceTemplate.ValueBool())

	var pullRequest RepositoryPullRequestModel
	diags = spec.PullRequest.As(ctx, &pullRequest, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "Update {{title}}", pullRequest.TitleTemplate.ValueString())
	require.False(t, pullRequest.EnforceTemplate.ValueBool())

	var commit RepositoryCommitModel
	diags = spec.Commit.As(ctx, &commit, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "Save {{resourceKind}}: {{title}}", commit.SingleResourceMessageTemplate.ValueString())
	require.True(t, commit.EnforceTemplate.ValueBool())
	require.Equal(t, "Grafana Bot", commit.SignerName.ValueString())
	require.Equal(t, "bot@example.com", commit.SignerEmail.ValueString())
	require.Equal(t, string(provisioningv0alpha1.GPGSigningMethod), commit.SigningMethod.ValueString())
	require.True(t, commit.SMIMECertificate.IsNull())
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
		"github_enterprise": types.ObjectNull(repositoryGitHubEnterpriseType.AttrTypes),
		"git":               types.ObjectNull(repositoryGitType.AttrTypes),
		"bitbucket":         types.ObjectNull(repositoryBitbucketType.AttrTypes),
		"gitlab":            types.ObjectNull(repositoryGitLabType.AttrTypes),
		"local":             types.ObjectNull(repositoryLocalType.AttrTypes),
		"connection":        types.ObjectNull(repositoryConnectionType.AttrTypes),
		"webhook": types.ObjectValueMust(repositoryWebhookType.AttrTypes, map[string]attr.Value{
			"base_url": types.StringValue("https://hooks.example.com"),
		}),
		"branch":       types.ObjectNull(repositoryBranchType.AttrTypes),
		"pull_request": types.ObjectNull(repositoryPullRequestType.AttrTypes),
		"commit":       types.ObjectNull(repositoryCommitType.AttrTypes),
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
