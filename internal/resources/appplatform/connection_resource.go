package appplatform

import (
	"context"
	"fmt"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	provisioningv0alpha1 "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
	connectionAPIGroup   = provisioningv0alpha1.GROUP
	connectionAPIVersion = provisioningv0alpha1.VERSION
	connectionKind       = "Connection"
)

type ProvisioningConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              provisioningv0alpha1.ConnectionSpec `json:"spec"`
	secureSubresourceSupport[provisioningv0alpha1.ConnectionSecure]
}

type ProvisioningConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ProvisioningConnection `json:"items"`
}

func (o *ProvisioningConnection) GetSpec() any {
	return o.Spec
}

func (o *ProvisioningConnection) SetSpec(spec any) error {
	cast, ok := spec.(provisioningv0alpha1.ConnectionSpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type ConnectionSpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *ProvisioningConnection) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     connectionAPIGroup,
		Version:   connectionAPIVersion,
		Kind:      connectionKind,
	}
}

func (o *ProvisioningConnection) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *ProvisioningConnection) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *ProvisioningConnection) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *ProvisioningConnection) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *ProvisioningConnection) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func (o *ProvisioningConnectionList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *ProvisioningConnectionList) SetItems(items []sdkresource.Object) {
	o.Items = make([]ProvisioningConnection, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*ProvisioningConnection)
	}
}

func (o *ProvisioningConnectionList) Copy() sdkresource.ListObject {
	cpy := &ProvisioningConnectionList{
		TypeMeta: o.TypeMeta,
		Items:    make([]ProvisioningConnection, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*ProvisioningConnection); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *ProvisioningConnectionList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func ConnectionKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			connectionAPIGroup,
			connectionAPIVersion,
			&ProvisioningConnection{},
			&ProvisioningConnectionList{},
			sdkresource.WithKind(connectionKind),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: sdkresource.NewJSONCodec(),
		},
	}
}

var connectionGitHubType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"app_id":          types.StringType,
		"installation_id": types.StringType,
	},
}

var connectionSpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":       types.StringType,
		"description": types.StringType,
		"type":        types.StringType,
		"url":         types.StringType,
		"github":      connectionGitHubType,
	},
}

type ConnectionSpecModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	URL         types.String `tfsdk:"url"`
	GitHub      types.Object `tfsdk:"github"`
}

type ConnectionGitHubModel struct {
	AppID          types.String `tfsdk:"app_id"`
	InstallationID types.String `tfsdk:"installation_id"`
}

func Connection() NamedResource {
	return NewNamedResource[*ProvisioningConnection, *ProvisioningConnectionList](
		common.CategoryGrafanaApps,
		ResourceConfig[*ProvisioningConnection]{
			Kind: ConnectionKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Git Sync connections.",
				MarkdownDescription: `
Manages Grafana Git Sync connections used by repositories for provider authentication.
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "Display name shown in the UI.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "Connection description.",
					},
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Connection provider type.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(provisioningv0alpha1.GithubConnectionType),
							),
						},
					},
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "Provider URL.",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"github": schema.SingleNestedBlock{
						Description: "GitHub App configuration.",
						Attributes: map[string]schema.Attribute{
							"app_id": schema.StringAttribute{
								Required:    true,
								Description: "GitHub App ID.",
							},
							"installation_id": schema.StringAttribute{
								Required:    true,
								Description: "GitHub App installation ID.",
							},
						},
					},
				},
				SecureValueAttributes: map[string]SecureValueAttribute{
					"private_key": {
						Optional:    true,
						APIName:     "privateKey",
						Description: "Private key for GitHub App authentication.",
					},
					"client_secret": {
						Optional:    true,
						APIName:     "clientSecret",
						Description: "Client secret.",
					},
					"token": {
						Optional:    true,
						Description: "Access token.",
					},
				},
			},
			SpecParser:   parseConnectionSpec,
			SpecSaver:    saveConnectionSpec,
			SecureParser: DefaultSecureParser[*ProvisioningConnection],
		},
	)
}

func parseConnectionSpec(ctx context.Context, src types.Object, dst *ProvisioningConnection) diag.Diagnostics {
	var data ConnectionSpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	validationDiags := validateConnectionSpecModel(data)
	if validationDiags.HasError() {
		return validationDiags
	}

	spec := provisioningv0alpha1.ConnectionSpec{
		Title: data.Title.ValueString(),
		Type:  provisioningv0alpha1.ConnectionType(data.Type.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		spec.Description = data.Description.ValueString()
	}
	if !data.URL.IsNull() && !data.URL.IsUnknown() {
		spec.URL = data.URL.ValueString()
	}

	if !data.GitHub.IsNull() && !data.GitHub.IsUnknown() {
		cfg, d := parseConnectionGitHub(ctx, data.GitHub)
		if d.HasError() {
			return d
		}
		spec.GitHub = &cfg
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return nil
}

func validateConnectionSpecModel(data ConnectionSpecModel) diag.Diagnostics {
	var diags diag.Diagnostics

	providerBlocks := map[provisioningv0alpha1.ConnectionType]types.Object{
		provisioningv0alpha1.GithubConnectionType: data.GitHub,
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
			"Invalid connection provider configuration",
			"Exactly one provider block must be configured: `github`.",
		)
	}

	selectedType := provisioningv0alpha1.ConnectionType(data.Type.ValueString())
	selectedBlock, found := providerBlocks[selectedType]
	if !found {
		diags.AddError(
			"Invalid connection provider type",
			fmt.Sprintf("unsupported `type` value %q", selectedType),
		)
		return diags
	}
	if selectedBlock.IsNull() || selectedBlock.IsUnknown() {
		diags.AddError(
			"Invalid connection provider configuration",
			fmt.Sprintf("`type = %q` requires the `%s` block to be configured.", selectedType, selectedType),
		)
	}

	return diags
}

func parseConnectionGitHub(ctx context.Context, src types.Object) (provisioningv0alpha1.GitHubConnectionConfig, diag.Diagnostics) {
	var data ConnectionGitHubModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return provisioningv0alpha1.GitHubConnectionConfig{}, d
	}

	return provisioningv0alpha1.GitHubConnectionConfig{
		AppID:          data.AppID.ValueString(),
		InstallationID: data.InstallationID.ValueString(),
	}, nil
}

func saveConnectionSpec(ctx context.Context, src *ProvisioningConnection, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	values["title"] = types.StringValue(src.Spec.Title)
	if src.Spec.Description != "" {
		values["description"] = types.StringValue(src.Spec.Description)
	} else {
		values["description"] = types.StringNull()
	}
	values["type"] = types.StringValue(string(src.Spec.Type))
	if src.Spec.URL != "" {
		values["url"] = types.StringValue(src.Spec.URL)
	} else {
		values["url"] = types.StringNull()
	}

	githubValue, d := saveConnectionGitHubSpec(ctx, src.Spec.GitHub)
	if d.HasError() {
		return d
	}
	values["github"] = githubValue

	spec, d := types.ObjectValue(connectionSpecType.AttrTypes, values)
	if d.HasError() {
		return d
	}
	dst.Spec = spec

	return nil
}

func saveConnectionGitHubSpec(ctx context.Context, src *provisioningv0alpha1.GitHubConnectionConfig) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(connectionGitHubType.AttrTypes), nil
	}

	return types.ObjectValueFrom(ctx, connectionGitHubType.AttrTypes, ConnectionGitHubModel{
		AppID:          types.StringValue(src.AppID),
		InstallationID: types.StringValue(src.InstallationID),
	})
}
