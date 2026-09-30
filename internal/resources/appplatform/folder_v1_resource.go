package appplatform

import (
	"context"
	"fmt"
	"regexp"

	folderv1 "github.com/grafana/grafana/apps/folder/pkg/apis/folder/v1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// apiVersionPattern requires the Kubernetes `group/version` form. Without it, a value such as
// "iam.grafana.app" with the version left off parses as group "" and would be sent to the API
// only to be rejected there; matching it here fails at plan time with a clearer message.
var apiVersionPattern = regexp.MustCompile(`^[^/]+/[^/]+$`)

// ownerReferenceTeamKind is the only owner kind Grafana does anything with, and so the only kind
// this resource manages: configuration is restricted to it, and reading a folder owned by
// anything else is an error rather than a silent removal. See ownerReferencesToModel.
const ownerReferenceTeamKind = "Team"

// FolderSpecModel is a Terraform model for a Grafana folder spec.
type FolderSpecModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
}

// OwnerReferenceModel is a Terraform model for a single Kubernetes owner reference.
//
// The Kubernetes `uid` field is deliberately absent. Grafana requires it to be non-empty but
// never checks it against the owner object — ownership is keyed on the name — so the Grafana UI
// simply sends the name for both. Exposing it would mean an Optional+Computed attribute whose
// value Terraform carries forward from prior state, which would silently keep the old team's uid
// when the name is repointed at a different team.
type OwnerReferenceModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}

// ownerReferenceAttrTypes is the attribute type map for one owner_references entry.
func ownerReferenceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"api_version": types.StringType,
		"kind":        types.StringType,
		"name":        types.StringType,
	}
}

func ownerReferenceObjectType() attr.Type {
	return types.ObjectType{AttrTypes: ownerReferenceAttrTypes()}
}

var folderSpecAttrTypes = map[string]attr.Type{
	"title":       types.StringType,
	"description": types.StringType,
}

// FolderV1 creates a new Grafana Folder v1 resource.
func FolderV1() NamedResource {
	return NewNamedResource[*folderv1.Folder, *folderv1.FolderList](
		common.CategoryGrafanaApps,
		ResourceConfig[*folderv1.Folder]{
			Kind: folderv1.FolderKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana folders.",
				MarkdownDescription: `
Manages Grafana folders using the new Grafana APIs.

Unlike ` + "`grafana_folder`" + `, this resource can assign folder ownership to a team via
` + "`metadata.owner_references`" + ` ("team folders"). Team ownership affects how the folder is
labelled and grouped in the UI; it does not grant any permissions on the folder.

Nest a folder under a parent by setting ` + "`metadata.folder_uid`" + `.

Requires Grafana 13.0 or later. Team folders are enabled by default from Grafana 13.1 — on
13.0.x the ` + "`teamFolders`" + ` feature toggle must be enabled, otherwise the owner reference is
stored but has no visible effect (no "Owned by" label and no Team folders grouping).

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [Team folders](https://grafana.com/docs/grafana/latest/administration/team-management/team-folders/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The title of the folder.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "The description of the folder.",
					},
				},
				MetadataBlocks: map[string]schema.Block{
					"owner_references": schema.ListNestedBlock{
						Description: "Kubernetes owner references for the folder. Declare a block with " +
							"`api_version = \"iam.grafana.app/v0alpha1\"` and `kind = \"Team\"` to make this a team " +
							"folder. Team ownership only affects how the folder is presented in the UI; it does not " +
							"grant permissions — use `grafana_folder_permission_item` for that. Grafana's own UI " +
							"assigns at most one owner per folder, though the API accepts several. Owner references " +
							"cannot be set on folders managed by a provisioning repository. Requires Grafana 13.1 " +
							"or later, or the `teamFolders` feature toggle on 13.0.x.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"api_version": schema.StringAttribute{
									Required:    true,
									Description: "The API version of the owner, in `group/version` form, e.g. `iam.grafana.app/v0alpha1`.",
									Validators: []validator.String{
										stringvalidator.RegexMatches(apiVersionPattern, "must be in `group/version` form, e.g. `iam.grafana.app/v0alpha1`"),
									},
								},
								"kind": schema.StringAttribute{
									Required: true,
									Description: "The kind of the owner. Only `Team` is supported — Grafana assigns folder " +
										"ownership to teams and to nothing else today. Reading a folder owned by anything " +
										"else fails with an error rather than removing that owner on the next apply.",
									Validators: []validator.String{
										// The API itself enforces no allow-list, so this is stricter than the
										// server on purpose: any other kind is accepted and then does nothing,
										// which is worth catching at plan time rather than silently shipping.
										stringvalidator.OneOf(ownerReferenceTeamKind),
									},
								},
								"name": schema.StringAttribute{
									Required:    true,
									Description: "The name of the owner object. For a team this is its UID, e.g. `grafana_team.my_team.team_uid`.",
								},
							},
						},
					},
				},
				OptionsAttributes: map[string]schema.Attribute{
					"allow_ui_updates": schema.BoolAttribute{
						Optional:    true,
						Description: "Set to true to allow editing the resource from the Grafana UI. By default, resources managed by Terraform cannot be edited in the UI. Enabling this option will cause divergence between the Terraform configuration and the resource in Grafana.",
					},
				},
			},
			SpecParser: func(ctx context.Context, src types.Object, dst *folderv1.Folder) diag.Diagnostics {
				var data FolderSpecModel
				if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				res := folderv1.FolderSpec{
					Title: data.Title.ValueString(),
				}

				if !data.Description.IsNull() && !data.Description.IsUnknown() {
					desc := data.Description.ValueString()
					res.Description = &desc
				}

				if err := dst.SetSpec(res); err != nil {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to set spec", err.Error()),
					}
				}

				return diag.Diagnostics{}
			},
			SpecSaver: func(ctx context.Context, src *folderv1.Folder, dst *ResourceModel) diag.Diagnostics {
				data := FolderSpecModel{
					Title: types.StringValue(src.Spec.Title),
					// Keep a nil description null rather than collapsing it to "", which would
					// diff against a configuration that omits `description`.
					Description: types.StringNull(),
				}
				if src.Spec.Description != nil {
					data.Description = types.StringValue(*src.Spec.Description)
				}

				spec, diags := types.ObjectValueFrom(ctx, folderSpecAttrTypes, &data)
				if diags.HasError() {
					return diags
				}
				dst.Spec = spec

				return diag.Diagnostics{}
			},
			MetadataParser: func(ctx context.Context, metadata types.Object, dst *folderv1.Folder) diag.Diagnostics {
				list, ok := metadata.Attributes()["owner_references"].(types.List)
				if !ok {
					return diag.Diagnostics{}
				}

				refs, diags := ownerReferencesFromModel(ctx, list)
				if diags.HasError() {
					return diags
				}

				// Set unconditionally. Update builds a fresh object from the Terraform model and
				// replaces the stored one, so a nil slice is how "ownership removed" is expressed.
				dst.SetOwnerReferences(refs)

				return diag.Diagnostics{}
			},
			MetadataSaver: func(ctx context.Context, src *folderv1.Folder, dst map[string]attr.Value) diag.Diagnostics {
				refs, diags := ownerReferencesToModel(ctx, src.GetName(), src.GetOwnerReferences())
				if diags.HasError() {
					return diags
				}
				dst["owner_references"] = refs

				return diag.Diagnostics{}
			},
		})
}

// ownerReferencesFromModel converts the configured owner_references list into Kubernetes owner
// references. A null or unknown list yields no references.
func ownerReferencesFromModel(ctx context.Context, src types.List) ([]metav1.OwnerReference, diag.Diagnostics) {
	var diags diag.Diagnostics

	if src.IsNull() || src.IsUnknown() {
		return nil, diags
	}

	var models []OwnerReferenceModel
	diags.Append(src.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	refs := make([]metav1.OwnerReference, 0, len(models))
	for _, m := range models {
		name := m.Name.ValueString()

		refs = append(refs, metav1.OwnerReference{
			APIVersion: m.APIVersion.ValueString(),
			Kind:       m.Kind.ValueString(),
			Name:       name,
			// Kubernetes requires a non-empty uid. Grafana does not resolve it against the
			// owner object, so send the name, exactly as the Grafana UI does.
			UID: k8stypes.UID(name),
		})
	}

	return refs, diags
}

// ownerReferencesToModel converts an object's Kubernetes owner references into the Terraform
// owner_references value.
//
// An owner whose kind is not `Team` is reported as an error rather than read into state. Such an
// owner cannot be expressed in configuration, because the `kind` attribute only accepts `Team`,
// so reading it in would leave the folder permanently diffed against its own configuration and
// the next apply would strip the owner with no way to opt out. Failing loudly leaves the folder
// unmanageable until the owner is removed, which is preferable to deleting a link Terraform did
// not create.
//
// Note the check is on the kind, not the API group. `api_version` is only shape-checked, so
// another group paired with `kind = "Team"` is configurable and must keep round-tripping --
// narrowing this further would reintroduce the config/state asymmetry that fails an apply with
// "Provider produced inconsistent result after apply".
//
// owner_references is a nested block, so a configuration declaring none of them is an empty list
// rather than null; returning an empty list here is what makes an unowned folder match it.
func ownerReferencesToModel(ctx context.Context, name string, src []metav1.OwnerReference) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	models := make([]OwnerReferenceModel, 0, len(src))
	for _, ref := range src {
		if ref.Kind != ownerReferenceTeamKind {
			diags.AddError(
				"Unsupported folder owner",
				fmt.Sprintf(
					"Folder %q is owned by %s/%s %q. This resource can only manage %s owners, so it "+
						"cannot manage this folder without removing that owner reference. Remove the owner "+
						"reference from the folder, or manage the folder outside Terraform.",
					name, ref.APIVersion, ref.Kind, ref.Name, ownerReferenceTeamKind,
				),
			)
			continue
		}

		models = append(models, OwnerReferenceModel{
			APIVersion: types.StringValue(ref.APIVersion),
			Kind:       types.StringValue(ref.Kind),
			Name:       types.StringValue(ref.Name),
		})
	}

	if diags.HasError() {
		return types.ListNull(ownerReferenceObjectType()), diags
	}

	list, listDiags := types.ListValueFrom(ctx, ownerReferenceObjectType(), models)
	diags.Append(listDiags...)

	return list, diags
}
