package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/dashboard-linter/lint"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1alpha1"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/appplatform/client"
)

// DashboardSpecModel is a model for the dashboard spec.
type DashboardSpecModel struct {
	JSON  jsontypes.Normalized `tfsdk:"json"`
	Title types.String         `tfsdk:"title"`
	Tags  types.List           `tfsdk:"tags"`
}

// Dashboard creates a new Grafana Dashboard resource.
func Dashboard() resource.Resource {
	return NewResource(ResourceConfig[*v1alpha1.Dashboard, *v1alpha1.DashboardList, v1alpha1.DashboardSpec]{
		Schema: ResourceSpecSchema{
			Description: "Manages Grafana dashboards.",
			MarkdownDescription: `
	Manages Grafana dashboards.

	* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
	* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
	`,
			SpecAttributes: map[string]schema.Attribute{
				"json": schema.StringAttribute{
					Required:    true,
					Description: "The JSON representation of the dashboard spec.",
					CustomType:  jsontypes.NormalizedType{},
					Validators: []validator.String{
						&DashboardJSONValidator{},
					},
				},
				"title": schema.StringAttribute{
					Optional:    true,
					Description: "The title of the dashboard. If not set, the title will be derived from the JSON spec.",
				},
				"tags": schema.ListAttribute{
					Optional:    true,
					Description: "The tags of the dashboard. If not set, the tags will be derived from the JSON spec.",
					ElementType: types.StringType,
				},
			},
		},
		Kind: v1alpha1.DashboardKind(),
		NewClientFn: func(reg client.Registry, stackOrOrgID int64, isOrg bool) (*client.NamespacedClient[*v1alpha1.Dashboard, *v1alpha1.DashboardList], error) {
			cli, err := reg.ClientFor(v1alpha1.DashboardKind())
			if err != nil {
				return nil, err
			}

			return client.NewNamespaced(
				client.NewResourceClient[*v1alpha1.Dashboard, *v1alpha1.DashboardList](cli, v1alpha1.DashboardKind()),
				stackOrOrgID, isOrg,
			), nil
		},
		SpecParser: func(ctx context.Context, spec types.Object, dst *v1alpha1.Dashboard) diag.Diagnostics {
			var data DashboardSpecModel
			if diag := spec.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}

			var res v1alpha1.DashboardSpec
			if diag := data.JSON.Unmarshal(&res); diag.HasError() {
				return diag
			}

			if !data.Title.IsNull() && !data.Title.IsUnknown() {
				res.Object["title"] = data.Title.ValueString()
			}

			if tags, ok := getTagsFromModel(data); ok {
				res.Object["tags"] = tags
			}

			delete(res.Object, "version")

			if err := dst.SetSpec(res); err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("failed to set spec", err.Error()),
				}
			}

			return diag.Diagnostics{}
		},
		SpecSaver: func(ctx context.Context, obj *v1alpha1.Dashboard, dst *ResourceModel) diag.Diagnostics {
			// HACK: for v0 we need to clean a few fields from the spec,
			// which are not supposed to be set by the user.
			delete(obj.Spec.Object, "version")

			json, err := json.Marshal(obj.Spec.Object)
			if err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("failed to marshal dashboard spec", err.Error()),
				}
			}

			var data DashboardSpecModel
			if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			}); diag.HasError() {
				return diag
			}
			data.JSON = jsontypes.NewNormalizedValue(string(json))

			// Only copy title from JSON if it is also set in Terraform.
			if !data.Title.IsNull() && !data.Title.IsUnknown() {
				tval := obj.Spec.Object["title"]
				title, ok := tval.(string)
				if !ok {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to get title", "title is not a string"),
					}
				}
				data.Title = types.StringValue(title)
			} else {
				data.Title = types.StringNull()
			}

			// Only copy tags from JSON if they are also set in Terraform.
			if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
				tags, diags := types.ListValueFrom(ctx, types.StringType, getTagsFromResource(obj))
				if diags.HasError() {
					return diags
				}
				data.Tags = tags
			} else {
				data.Tags = types.ListNull(types.StringType)
			}

			spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"json":  types.StringType,
				"title": types.StringType,
				"tags":  types.ListType{ElemType: types.StringType},
			}, &data)
			if diags.HasError() {
				return diags
			}
			dst.Spec = spec

			return diag.Diagnostics{}
		},
	})
}

// DashboardJSONValidator is a validator for the dashboard spec.
type DashboardJSONValidator struct{}

// Description returns the description of the validator.
func (v *DashboardJSONValidator) Description(ctx context.Context) string {
	return "validates the dashboard spec"
}

// MarkdownDescription returns the markdown description of the validator.
func (v *DashboardJSONValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString validates the dashboard spec.
func (v *DashboardJSONValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	var data ResourceModel
	if diag := req.Config.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var opts ResourceOptions
	if diag := ParseResourceOptionsFromModel(ctx, data, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	bytes := []byte(req.ConfigValue.ValueString())

	if opts.Validate {
		if err := ValidateDashboard(bytes); err != nil {
			resp.Diagnostics.AddError("failed to validate dashboard", err.Error())
			return
		}
	}

	if len(opts.LintRules) > 0 {
		results, ok, err := LintDashboard(GetLintRulesForNames(opts.LintRules), bytes)
		if err != nil {
			resp.Diagnostics.AddError("failed to lint dashboard", err.Error())
			return
		}

		if ok {
			return
		}

		if results.Warnings != "" {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("spec").AtName("json"),
				"dashboard linter returned warnings",
				results.Warnings,
			)
		}

		if results.Errors != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("json"),
				"dashboard linter returned errors",
				results.Errors,
			)
		}
	}
}

// ValidateDashboard validates the dashboard spec.
func ValidateDashboard(bspec []byte) error {
	var dash dashboard.Dashboard
	return dash.UnmarshalJSONStrict(bspec)
}

// DashboardLintRules is a list of all the lint rules.
var DashboardLintRules = []lint.Rule{
	lint.NewTemplateDatasourceRule(),
	lint.NewTemplateJobRule(),
	lint.NewTemplateInstanceRule(),
	lint.NewTemplateLabelPromQLRule(),
	lint.NewTemplateOnTimeRangeReloadRule(),
	lint.NewPanelDatasourceRule(),
	lint.NewPanelTitleDescriptionRule(),
	lint.NewPanelUnitsRule(),
	lint.NewPanelNoTargetsRule(),
	lint.NewTargetLogQLRule(),
	lint.NewTargetLogQLAutoRule(),
	lint.NewTargetPromQLRule(),
	lint.NewTargetRateIntervalRule(),
	lint.NewTargetJobRule(),
	lint.NewTargetInstanceRule(),
	lint.NewTargetCounterAggRule(),
	lint.NewUneditableRule(),
}

// GetLintRulesForNames returns a list of lint rules for the given rule names.
func GetLintRulesForNames(names []string) []lint.Rule {
	res := make([]lint.Rule, 0, len(names))

	for _, name := range names {
		for _, rule := range DashboardLintRules {
			if rule.Name() == name {
				res = append(res, rule)
			}
		}
	}

	return res
}

// LintResults is the result of linting a dashboard.
type LintResults struct {
	Warnings string
	Errors   string
}

// LintDashboard lints a dashboard.
func LintDashboard(rules []lint.Rule, spec []byte) (LintResults, bool, error) {
	dash, err := lint.NewDashboard(spec)
	if err != nil {
		return LintResults{}, false, err
	}

	resSet := &lint.ResultSet{}
	for _, r := range rules {
		r.Lint(dash, resSet)
	}

	var (
		warnings strings.Builder
		errors   strings.Builder
	)

	var anyerr bool
	for _, r := range resSet.ByRule() {
		for _, rc := range r {
			for _, r := range rc.Result.Results {
				if r.Severity == lint.Error {
					errors.WriteString(fmt.Sprintf("\t* %s\n", r.Message))
					anyerr = true
				}

				if r.Severity == lint.Warning {
					warnings.WriteString(fmt.Sprintf("\t* %s\n", r.Message))
					anyerr = true
				}
			}
		}
	}

	return LintResults{
		Warnings: warnings.String(),
		Errors:   errors.String(),
	}, !anyerr, nil
}

func getTagsFromResource(src *v1alpha1.Dashboard) []string {
	tags, ok := src.Spec.Object["tags"]
	if !ok {
		return nil
	}

	taglist, ok := tags.([]any)
	if !ok {
		return nil
	}

	if taglist == nil {
		return nil
	}

	res := make([]string, 0, len(taglist))
	for _, tag := range taglist {
		if tag, ok := tag.(string); ok {
			res = append(res, tag)
		}
	}

	return res
}

func getTagsFromModel(data DashboardSpecModel) ([]string, bool) {
	if data.Tags.IsNull() || data.Tags.IsUnknown() {
		return nil, false
	}

	tags := make([]string, 0, len(data.Tags.Elements()))
	for _, tag := range data.Tags.Elements() {
		if tag, ok := tag.(types.String); ok {
			tags = append(tags, tag.ValueString())
		}
	}

	return tags, true
}
