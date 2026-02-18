package appplatform

import (
	"context"
	"fmt"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	provisioningv0alpha1 "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	repositoryAPIGroup   = provisioningv0alpha1.GROUP
	repositoryAPIVersion = provisioningv0alpha1.VERSION
	repositoryKind       = "Repository"
)

type ProvisioningRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              provisioningv0alpha1.RepositorySpec `json:"spec"`
	secureSubresourceSupport[provisioningv0alpha1.SecureValues]
}

type ProvisioningRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ProvisioningRepository `json:"items"`
}

type Workflow = provisioningv0alpha1.Workflow

const (
	WorkflowWrite  Workflow = provisioningv0alpha1.WriteWorkflow
	WorkflowBranch Workflow = provisioningv0alpha1.BranchWorkflow
)

type SyncTarget = provisioningv0alpha1.SyncTargetType

const (
	SyncTargetInstance SyncTarget = provisioningv0alpha1.SyncTargetTypeInstance
	SyncTargetFolder   SyncTarget = provisioningv0alpha1.SyncTargetTypeFolder
)

type RepositoryType = provisioningv0alpha1.RepositoryType

const (
	RepositoryTypeLocal     RepositoryType = provisioningv0alpha1.LocalRepositoryType
	RepositoryTypeGitHub    RepositoryType = provisioningv0alpha1.GitHubRepositoryType
	RepositoryTypeGit       RepositoryType = provisioningv0alpha1.GitRepositoryType
	RepositoryTypeBitbucket RepositoryType = provisioningv0alpha1.BitbucketRepositoryType
	RepositoryTypeGitLab    RepositoryType = provisioningv0alpha1.GitLabRepositoryType
)

type RepositorySpec = provisioningv0alpha1.RepositorySpec
type SyncOptions = provisioningv0alpha1.SyncOptions
type LocalRepositoryConfig = provisioningv0alpha1.LocalRepositoryConfig
type GitHubRepositoryConfig = provisioningv0alpha1.GitHubRepositoryConfig
type GitRepositoryConfig = provisioningv0alpha1.GitRepositoryConfig
type BitbucketRepositoryConfig = provisioningv0alpha1.BitbucketRepositoryConfig
type GitLabRepositoryConfig = provisioningv0alpha1.GitLabRepositoryConfig
type ConnectionInfo = provisioningv0alpha1.ConnectionInfo

func (o *ProvisioningRepository) GetSpec() any {
	return o.Spec
}

func (o *ProvisioningRepository) SetSpec(spec any) error {
	cast, ok := spec.(RepositorySpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type RepositorySpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *ProvisioningRepository) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     repositoryAPIGroup,
		Version:   repositoryAPIVersion,
		Kind:      repositoryKind,
	}
}

func (o *ProvisioningRepository) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *ProvisioningRepository) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *ProvisioningRepository) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *ProvisioningRepository) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *ProvisioningRepository) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func (o *ProvisioningRepositoryList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *ProvisioningRepositoryList) SetItems(items []sdkresource.Object) {
	o.Items = make([]ProvisioningRepository, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*ProvisioningRepository)
	}
}

func (o *ProvisioningRepositoryList) Copy() sdkresource.ListObject {
	cpy := &ProvisioningRepositoryList{
		TypeMeta: o.TypeMeta,
		Items:    make([]ProvisioningRepository, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*ProvisioningRepository); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *ProvisioningRepositoryList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func RepositoryKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			repositoryAPIGroup,
			repositoryAPIVersion,
			&ProvisioningRepository{},
			&ProvisioningRepositoryList{},
			sdkresource.WithKind(repositoryKind),
			sdkresource.WithPlural("repositories"),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: sdkresource.NewJSONCodec(),
		},
	}
}

var repositorySyncType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"enabled":          types.BoolType,
		"target":           types.StringType,
		"interval_seconds": types.Int64Type,
	},
}

var repositoryGitHubType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"url":                         types.StringType,
		"branch":                      types.StringType,
		"path":                        types.StringType,
		"generate_dashboard_previews": types.BoolType,
	},
}

var repositoryGitType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"url":        types.StringType,
		"branch":     types.StringType,
		"token_user": types.StringType,
		"path":       types.StringType,
	},
}

var repositoryBitbucketType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"url":        types.StringType,
		"branch":     types.StringType,
		"token_user": types.StringType,
		"path":       types.StringType,
	},
}

var repositoryGitLabType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"url":    types.StringType,
		"branch": types.StringType,
		"path":   types.StringType,
	},
}

var repositoryLocalType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"path": types.StringType,
	},
}

var repositoryConnectionType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name": types.StringType,
	},
}

var repositorySpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":       types.StringType,
		"description": types.StringType,
		"workflows":   types.ListType{ElemType: types.StringType},
		"sync":        repositorySyncType,
		"type":        types.StringType,
		"github":      repositoryGitHubType,
		"git":         repositoryGitType,
		"bitbucket":   repositoryBitbucketType,
		"gitlab":      repositoryGitLabType,
		"local":       repositoryLocalType,
		"connection":  repositoryConnectionType,
	},
}

type RepositorySpecModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Workflows   types.List   `tfsdk:"workflows"`
	Sync        types.Object `tfsdk:"sync"`
	Type        types.String `tfsdk:"type"`
	GitHub      types.Object `tfsdk:"github"`
	Git         types.Object `tfsdk:"git"`
	Bitbucket   types.Object `tfsdk:"bitbucket"`
	GitLab      types.Object `tfsdk:"gitlab"`
	Local       types.Object `tfsdk:"local"`
	Connection  types.Object `tfsdk:"connection"`
}

type RepositorySyncModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Target          types.String `tfsdk:"target"`
	IntervalSeconds types.Int64  `tfsdk:"interval_seconds"`
}

type RepositoryGitHubModel struct {
	URL                       types.String `tfsdk:"url"`
	Branch                    types.String `tfsdk:"branch"`
	Path                      types.String `tfsdk:"path"`
	GenerateDashboardPreviews types.Bool   `tfsdk:"generate_dashboard_previews"`
}

type RepositoryGitModel struct {
	URL       types.String `tfsdk:"url"`
	Branch    types.String `tfsdk:"branch"`
	TokenUser types.String `tfsdk:"token_user"`
	Path      types.String `tfsdk:"path"`
}

type RepositoryBitbucketModel struct {
	URL       types.String `tfsdk:"url"`
	Branch    types.String `tfsdk:"branch"`
	TokenUser types.String `tfsdk:"token_user"`
	Path      types.String `tfsdk:"path"`
}

type RepositoryGitLabModel struct {
	URL    types.String `tfsdk:"url"`
	Branch types.String `tfsdk:"branch"`
	Path   types.String `tfsdk:"path"`
}

type RepositoryLocalModel struct {
	Path types.String `tfsdk:"path"`
}

type RepositoryConnectionModel struct {
	Name types.String `tfsdk:"name"`
}

func Repository() NamedResource {
	return NewNamedResource[*ProvisioningRepository, *ProvisioningRepositoryList](
		common.CategoryGrafanaApps,
		ResourceConfig[*ProvisioningRepository]{
			Kind: RepositoryKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Git Sync repositories.",
				MarkdownDescription: `
Manages Grafana Git Sync repositories for provisioning dashboards and related resources.
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "Display name shown in the UI.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "Repository description.",
					},
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Repository provider type: local, github, git, bitbucket, or gitlab.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(RepositoryTypeLocal),
								string(RepositoryTypeGitHub),
								string(RepositoryTypeGit),
								string(RepositoryTypeBitbucket),
								string(RepositoryTypeGitLab),
							),
						},
					},
					"workflows": schema.ListAttribute{
						Optional:    true,
						Description: "Allowed change workflows: write, branch.",
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(
									string(WorkflowWrite),
									string(WorkflowBranch),
								),
							),
						},
					},
				},
				SpecBlocks: map[string]schema.Block{
					"sync": schema.SingleNestedBlock{
						Description: "Sync configuration.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Required:    true,
								Description: "Whether sync is enabled.",
							},
							"target": schema.StringAttribute{
								Required:    true,
								Description: "Sync target: instance or folder.",
								Validators: []validator.String{
									stringvalidator.OneOf(string(SyncTargetInstance), string(SyncTargetFolder)),
								},
							},
							"interval_seconds": schema.Int64Attribute{
								Optional:    true,
								Description: "Sync interval in seconds.",
							},
						},
					},
					"github": schema.SingleNestedBlock{
						Description: "GitHub repository configuration.",
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Optional:    true,
								Description: "Repository URL.",
							},
							"branch": schema.StringAttribute{
								Optional:    true,
								Description: "Branch to sync.",
							},
							"path": schema.StringAttribute{
								Optional:    true,
								Description: "Optional subdirectory path.",
							},
							"generate_dashboard_previews": schema.BoolAttribute{
								Optional:    true,
								Description: "Whether to generate dashboard previews.",
							},
						},
					},
					"git": schema.SingleNestedBlock{
						Description: "Generic git repository configuration.",
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Optional:    true,
								Description: "Repository URL.",
							},
							"branch": schema.StringAttribute{
								Optional:    true,
								Description: "Branch to sync.",
							},
							"token_user": schema.StringAttribute{
								Optional:    true,
								Description: "Username for PAT auth.",
							},
							"path": schema.StringAttribute{
								Optional:    true,
								Description: "Optional subdirectory path.",
							},
						},
					},
					"bitbucket": schema.SingleNestedBlock{
						Description: "Bitbucket repository configuration.",
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Optional:    true,
								Description: "Repository URL.",
							},
							"branch": schema.StringAttribute{
								Optional:    true,
								Description: "Branch to sync.",
							},
							"token_user": schema.StringAttribute{
								Optional:    true,
								Description: "Username for PAT auth.",
							},
							"path": schema.StringAttribute{
								Optional:    true,
								Description: "Optional subdirectory path.",
							},
						},
					},
					"gitlab": schema.SingleNestedBlock{
						Description: "GitLab repository configuration.",
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Optional:    true,
								Description: "Repository URL.",
							},
							"branch": schema.StringAttribute{
								Optional:    true,
								Description: "Branch to sync.",
							},
							"path": schema.StringAttribute{
								Optional:    true,
								Description: "Optional subdirectory path.",
							},
						},
					},
					"local": schema.SingleNestedBlock{
						Description: "Local filesystem repository configuration.",
						Attributes: map[string]schema.Attribute{
							"path": schema.StringAttribute{
								Optional:    true,
								Description: "Filesystem path.",
							},
						},
					},
					"connection": schema.SingleNestedBlock{
						Description: "Connection resource reference.",
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Optional:    true,
								Description: "Connection resource name.",
							},
						},
					},
				},
				SecureValueAttributes: map[string]SecureValueAttribute{
					"token": {
						Optional:    true,
						Description: "Token for repository authentication.",
					},
					"webhook_secret": {
						Optional:    true,
						APIName:     "webhookSecret",
						Description: "Webhook secret.",
					},
				},
			},
			SpecParser:   parseRepositorySpec,
			SpecSaver:    saveRepositorySpec,
			SecureParser: DefaultSecureParser[*ProvisioningRepository],
		},
	)
}

func parseRepositorySpec(ctx context.Context, src types.Object, dst *ProvisioningRepository) diag.Diagnostics {
	var data RepositorySpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	validationDiags := validateRepositorySpecModel(data)
	if validationDiags.HasError() {
		return validationDiags
	}

	spec := RepositorySpec{
		Title: data.Title.ValueString(),
		Type:  RepositoryType(data.Type.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		spec.Description = data.Description.ValueString()
	}

	if !data.Workflows.IsNull() && !data.Workflows.IsUnknown() {
		var workflows []string
		if d := data.Workflows.ElementsAs(ctx, &workflows, false); d.HasError() {
			return d
		}
		spec.Workflows = make([]Workflow, 0, len(workflows))
		for _, workflow := range workflows {
			spec.Workflows = append(spec.Workflows, Workflow(workflow))
		}
	}

	syncSpec, syncDiags := parseRepositorySync(ctx, data.Sync)
	if syncDiags.HasError() {
		return syncDiags
	}
	spec.Sync = syncSpec

	if !data.GitHub.IsNull() && !data.GitHub.IsUnknown() {
		cfg, d := parseRepositoryGitHub(ctx, data.GitHub)
		if d.HasError() {
			return d
		}
		spec.GitHub = &cfg
	}

	if !data.Git.IsNull() && !data.Git.IsUnknown() {
		cfg, d := parseRepositoryGit(ctx, data.Git)
		if d.HasError() {
			return d
		}
		spec.Git = &cfg
	}

	if !data.Bitbucket.IsNull() && !data.Bitbucket.IsUnknown() {
		cfg, d := parseRepositoryBitbucket(ctx, data.Bitbucket)
		if d.HasError() {
			return d
		}
		spec.Bitbucket = &cfg
	}

	if !data.GitLab.IsNull() && !data.GitLab.IsUnknown() {
		cfg, d := parseRepositoryGitLab(ctx, data.GitLab)
		if d.HasError() {
			return d
		}
		spec.GitLab = &cfg
	}

	if !data.Local.IsNull() && !data.Local.IsUnknown() {
		cfg, d := parseRepositoryLocal(ctx, data.Local)
		if d.HasError() {
			return d
		}
		spec.Local = &cfg
	}

	if !data.Connection.IsNull() && !data.Connection.IsUnknown() {
		cfg, d := parseRepositoryConnection(ctx, data.Connection)
		if d.HasError() {
			return d
		}
		spec.Connection = &cfg
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func validateRepositorySpecModel(data RepositorySpecModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.Sync.IsNull() || data.Sync.IsUnknown() {
		diags.AddError(
			"Invalid repository spec",
			"`sync` block is required.",
		)
	}

	providerBlocks := map[RepositoryType]types.Object{
		RepositoryTypeLocal:     data.Local,
		RepositoryTypeGitHub:    data.GitHub,
		RepositoryTypeGit:       data.Git,
		RepositoryTypeBitbucket: data.Bitbucket,
		RepositoryTypeGitLab:    data.GitLab,
	}

	configuredCount := 0
	for _, block := range providerBlocks {
		if block.IsNull() || block.IsUnknown() {
			continue
		}
		configuredCount++
	}

	if configuredCount != 1 {
		diags.AddError(
			"Invalid repository provider configuration",
			"Exactly one provider block must be configured: `local`, `github`, `git`, `bitbucket`, or `gitlab`.",
		)
	}

	selectedType := RepositoryType(data.Type.ValueString())
	selectedBlock, found := providerBlocks[selectedType]
	if !found {
		diags.AddError(
			"Invalid repository provider type",
			fmt.Sprintf("unsupported `type` value %q", selectedType),
		)
		return diags
	}
	if selectedBlock.IsNull() || selectedBlock.IsUnknown() {
		diags.AddError(
			"Invalid repository provider configuration",
			fmt.Sprintf("`type = %q` requires the `%s` block to be configured.", selectedType, selectedType),
		)
	}

	return diags
}

func parseRepositorySync(ctx context.Context, src types.Object) (SyncOptions, diag.Diagnostics) {
	var data RepositorySyncModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return SyncOptions{}, d
	}

	res := SyncOptions{
		Enabled: data.Enabled.ValueBool(),
		Target:  SyncTarget(data.Target.ValueString()),
	}
	if !data.IntervalSeconds.IsNull() && !data.IntervalSeconds.IsUnknown() {
		res.IntervalSeconds = data.IntervalSeconds.ValueInt64()
	}

	return res, nil
}

func parseRepositoryGitHub(ctx context.Context, src types.Object) (GitHubRepositoryConfig, diag.Diagnostics) {
	var data RepositoryGitHubModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return GitHubRepositoryConfig{}, d
	}

	res := GitHubRepositoryConfig{
		URL:    data.URL.ValueString(),
		Branch: data.Branch.ValueString(),
	}
	if !data.Path.IsNull() && !data.Path.IsUnknown() {
		res.Path = data.Path.ValueString()
	}
	if !data.GenerateDashboardPreviews.IsNull() && !data.GenerateDashboardPreviews.IsUnknown() {
		res.GenerateDashboardPreviews = data.GenerateDashboardPreviews.ValueBool()
	}

	return res, nil
}

func parseRepositoryGit(ctx context.Context, src types.Object) (GitRepositoryConfig, diag.Diagnostics) {
	var data RepositoryGitModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return GitRepositoryConfig{}, d
	}

	res := GitRepositoryConfig{
		URL:    data.URL.ValueString(),
		Branch: data.Branch.ValueString(),
	}
	if !data.TokenUser.IsNull() && !data.TokenUser.IsUnknown() {
		res.TokenUser = data.TokenUser.ValueString()
	}
	if !data.Path.IsNull() && !data.Path.IsUnknown() {
		res.Path = data.Path.ValueString()
	}

	return res, nil
}

func parseRepositoryBitbucket(ctx context.Context, src types.Object) (BitbucketRepositoryConfig, diag.Diagnostics) {
	var data RepositoryBitbucketModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return BitbucketRepositoryConfig{}, d
	}

	res := BitbucketRepositoryConfig{
		URL:    data.URL.ValueString(),
		Branch: data.Branch.ValueString(),
	}
	if !data.TokenUser.IsNull() && !data.TokenUser.IsUnknown() {
		res.TokenUser = data.TokenUser.ValueString()
	}
	if !data.Path.IsNull() && !data.Path.IsUnknown() {
		res.Path = data.Path.ValueString()
	}

	return res, nil
}

func parseRepositoryGitLab(ctx context.Context, src types.Object) (GitLabRepositoryConfig, diag.Diagnostics) {
	var data RepositoryGitLabModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return GitLabRepositoryConfig{}, d
	}

	res := GitLabRepositoryConfig{
		URL:    data.URL.ValueString(),
		Branch: data.Branch.ValueString(),
	}
	if !data.Path.IsNull() && !data.Path.IsUnknown() {
		res.Path = data.Path.ValueString()
	}

	return res, nil
}

func parseRepositoryLocal(ctx context.Context, src types.Object) (LocalRepositoryConfig, diag.Diagnostics) {
	var data RepositoryLocalModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return LocalRepositoryConfig{}, d
	}

	return LocalRepositoryConfig{Path: data.Path.ValueString()}, nil
}

func parseRepositoryConnection(ctx context.Context, src types.Object) (ConnectionInfo, diag.Diagnostics) {
	var data RepositoryConnectionModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return ConnectionInfo{}, d
	}

	return ConnectionInfo{Name: data.Name.ValueString()}, nil
}

func saveRepositorySpec(ctx context.Context, src *ProvisioningRepository, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	values["title"] = types.StringValue(src.Spec.Title)
	if src.Spec.Description != "" {
		values["description"] = types.StringValue(src.Spec.Description)
	} else {
		values["description"] = types.StringNull()
	}
	values["type"] = types.StringValue(string(src.Spec.Type))

	if len(src.Spec.Workflows) > 0 {
		workflowValues := make([]string, 0, len(src.Spec.Workflows))
		for _, workflow := range src.Spec.Workflows {
			workflowValues = append(workflowValues, string(workflow))
		}
		workflows, d := types.ListValueFrom(ctx, types.StringType, workflowValues)
		if d.HasError() {
			return d
		}
		values["workflows"] = workflows
	} else {
		values["workflows"] = types.ListNull(types.StringType)
	}

	syncModel := RepositorySyncModel{
		Enabled: types.BoolValue(src.Spec.Sync.Enabled),
		Target:  types.StringValue(string(src.Spec.Sync.Target)),
	}
	if src.Spec.Sync.IntervalSeconds > 0 {
		syncModel.IntervalSeconds = types.Int64Value(src.Spec.Sync.IntervalSeconds)
	} else {
		syncModel.IntervalSeconds = types.Int64Null()
	}
	syncValue, d := types.ObjectValueFrom(ctx, repositorySyncType.AttrTypes, syncModel)
	if d.HasError() {
		return d
	}
	values["sync"] = syncValue

	githubValue, d := saveRepositoryGitHubSpec(ctx, src.Spec.GitHub)
	if d.HasError() {
		return d
	}
	values["github"] = githubValue

	gitValue, d := saveRepositoryGitSpec(ctx, src.Spec.Git)
	if d.HasError() {
		return d
	}
	values["git"] = gitValue

	bitbucketValue, d := saveRepositoryBitbucketSpec(ctx, src.Spec.Bitbucket)
	if d.HasError() {
		return d
	}
	values["bitbucket"] = bitbucketValue

	gitlabValue, d := saveRepositoryGitLabSpec(ctx, src.Spec.GitLab)
	if d.HasError() {
		return d
	}
	values["gitlab"] = gitlabValue

	localValue, d := saveRepositoryLocalSpec(ctx, src.Spec.Local)
	if d.HasError() {
		return d
	}
	values["local"] = localValue

	connectionValue, d := saveRepositoryConnectionSpec(ctx, src.Spec.Connection)
	if d.HasError() {
		return d
	}
	values["connection"] = connectionValue

	spec, d := types.ObjectValue(repositorySpecType.AttrTypes, values)
	if d.HasError() {
		return d
	}
	dst.Spec = spec

	return nil
}

func saveRepositoryGitHubSpec(ctx context.Context, src *GitHubRepositoryConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryGitHubType.AttrTypes), nil
	}

	data := RepositoryGitHubModel{
		URL:    types.StringValue(src.URL),
		Branch: types.StringValue(src.Branch),
	}
	if src.Path != "" {
		data.Path = types.StringValue(src.Path)
	} else {
		data.Path = types.StringNull()
	}
	data.GenerateDashboardPreviews = types.BoolValue(src.GenerateDashboardPreviews)

	return types.ObjectValueFrom(ctx, repositoryGitHubType.AttrTypes, data)
}

func saveRepositoryGitSpec(ctx context.Context, src *GitRepositoryConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryGitType.AttrTypes), nil
	}

	data := RepositoryGitModel{
		URL:    types.StringValue(src.URL),
		Branch: types.StringValue(src.Branch),
	}
	if src.TokenUser != "" {
		data.TokenUser = types.StringValue(src.TokenUser)
	} else {
		data.TokenUser = types.StringNull()
	}
	if src.Path != "" {
		data.Path = types.StringValue(src.Path)
	} else {
		data.Path = types.StringNull()
	}

	return types.ObjectValueFrom(ctx, repositoryGitType.AttrTypes, data)
}

func saveRepositoryBitbucketSpec(ctx context.Context, src *BitbucketRepositoryConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryBitbucketType.AttrTypes), nil
	}

	data := RepositoryBitbucketModel{
		URL:    types.StringValue(src.URL),
		Branch: types.StringValue(src.Branch),
	}
	if src.TokenUser != "" {
		data.TokenUser = types.StringValue(src.TokenUser)
	} else {
		data.TokenUser = types.StringNull()
	}
	if src.Path != "" {
		data.Path = types.StringValue(src.Path)
	} else {
		data.Path = types.StringNull()
	}

	return types.ObjectValueFrom(ctx, repositoryBitbucketType.AttrTypes, data)
}

func saveRepositoryGitLabSpec(ctx context.Context, src *GitLabRepositoryConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryGitLabType.AttrTypes), nil
	}

	data := RepositoryGitLabModel{
		URL:    types.StringValue(src.URL),
		Branch: types.StringValue(src.Branch),
	}
	if src.Path != "" {
		data.Path = types.StringValue(src.Path)
	} else {
		data.Path = types.StringNull()
	}

	return types.ObjectValueFrom(ctx, repositoryGitLabType.AttrTypes, data)
}

func saveRepositoryLocalSpec(ctx context.Context, src *LocalRepositoryConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryLocalType.AttrTypes), nil
	}

	return types.ObjectValueFrom(ctx, repositoryLocalType.AttrTypes, RepositoryLocalModel{
		Path: types.StringValue(src.Path),
	})
}

func saveRepositoryConnectionSpec(ctx context.Context, src *ConnectionInfo) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(repositoryConnectionType.AttrTypes), nil
	}

	return types.ObjectValueFrom(ctx, repositoryConnectionType.AttrTypes, RepositoryConnectionModel{
		Name: types.StringValue(src.Name),
	})
}
