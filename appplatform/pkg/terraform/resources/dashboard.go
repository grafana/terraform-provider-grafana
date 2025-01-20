package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/terraform-provider-grafana/appplatform/pkg/client"
	"github.com/grafana/terraform-provider-grafana/appplatform/pkg/generated/resource/dashboard/v0alpha1"

	"github.com/grafana/dashboard-linter/lint"
	"github.com/grafana/grafana-app-sdk/k8s"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/client-go/rest"
)

// DashboardModel represents a Grafana dashboard Terraform model.
type DashboardModel struct {
	UUID      types.String `tfsdk:"uuid"`
	UID       types.String `tfsdk:"uid"`
	Title     types.String `tfsdk:"title"`
	FolderUID types.String `tfsdk:"folder_uid"`
	URL       types.String `tfsdk:"url"`
	Version   types.String `tfsdk:"version"`
	Spec      types.String `tfsdk:"spec"`
	Tags      types.List   `tfsdk:"tags"`
	Options   types.Object `tfsdk:"options"`
}

// DashboardModelOptions represents the options for a Grafana dashboard Terraform model.
type DashboardModelOptions struct {
	Overwrite types.Bool `tfsdk:"overwrite"`
	Validate  types.Bool `tfsdk:"validate"`
	LintRules types.List `tfsdk:"lint_rules"`
}

// DashboardOptions represents the options for applying a Grafana dashboard.
type DashboardOptions struct {
	Overwrite bool
	Validate  bool
	LintRules []string
}

// DashboardResource is a resource that manages Grafana dashboards.
type DashboardResource struct {
	client *client.NamespacedClient[*v0alpha1.Dashboard, *v0alpha1.DashboardList]
}

// NewDashboardResource creates a new DashboardResource.
func NewDashboardResource() resource.Resource {
	return &DashboardResource{}
}

// Schema returns the schema for the DashboardResource.
func (r *DashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		MarkdownDescription: `
	Manages Grafana dashboards.

	* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
	* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
	`,
		Attributes: map[string]schema.Attribute{
			// Required
			"uid": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of a dashboard, used to construct its URL. The uid allows having consistent URLs for accessing dashboards and when syncing dashboards between multiple Grafana installs.",
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The name of the dashboard, visible in the UI.",
			},
			"spec": schema.StringAttribute{
				Required:    true,
				Description: "The complete dashboard JSON.",
				PlanModifiers: []planmodifier.String{
					&DashboardNormalizer{},
				},
			},

			// Optional
			"folder_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The UID of the folder to save the dashboard in.",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags to attach to the dashboard. Tags can be used to filter dashboards in the Grafana UI.",
			},

			// Computed
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The globally unique identifier of a dashboard, used by the API for tracking.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The full URL of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Whenever you save a version of your dashboard, a copy of that version is saved so that previous versions of your dashboard are not lost.",
			},
		},
		Blocks: map[string]schema.Block{
			"options": schema.SingleNestedBlock{
				Description: "Options for applying the dashboard.",
				Attributes: map[string]schema.Attribute{
					"overwrite": schema.BoolAttribute{
						Optional:    true,
						Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
					},
					"validate": schema.BoolAttribute{
						Optional:    true,
						Description: "Set to true if you want to perform client-side validation before submitting the dashboard to Grafana server.",
					},
					"lint_rules": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "A list of lint rules to apply to the dashboard. Lint rules are used to validate the dashboard configuration.",
					},
				},
			},
		},
	}
}

// Metadata returns the metadata for the DashboardResource.
func (r *DashboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_dashboards_dashboard"
}

// Configure initializes the DashboardResource.
func (r *DashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// TODO: provide clients from `pkg/provider/framework_provider.go`, using `req.ProviderData`.
	tokenb, err := os.ReadFile(".token")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read token", err.Error())
		return
	}

	registry := k8s.NewClientRegistry(rest.Config{
		Host:        "https://localhost:3000",
		APIPath:     "/apis",
		BearerToken: strings.TrimSpace(string(tokenb)),
		UserAgent: fmt.Sprintf(
			"Terraform/%s (+https://www.terraform.io) terraform-provider-grafana/%s",
			"v1.10.1",
			"v0alpha1.0.1+testing",
		),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}, k8s.DefaultClientConfig())

	cli, err := v0alpha1.NewOrgClient(registry, 1)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Dashboard API client",
			err.Error(),
		)

		return
	}

	r.client = cli
}

// Create creates a new dashboard.
func (r *DashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DashboardModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var dash v0alpha1.Dashboard
	if diag := ParseDashboard(ctx, data, &dash); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Create(ctx, &dash, sdkresource.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dashboard", err.Error())
		return
	}

	if diag := SaveDashboardState(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the dashboard.
func (r *DashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DashboardModel
	if diag := req.Plan.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var opts DashboardOptions
	if diag := ParseOptions(ctx, data.Options, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var dash v0alpha1.Dashboard
	if diag := ParseDashboard(ctx, data, &dash); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	reqopts := sdkresource.UpdateOptions{
		ResourceVersion: dash.ResourceVersion,
	}

	if opts.Overwrite {
		reqopts.ResourceVersion = ""
	}

	res, err := r.client.Update(ctx, &dash, reqopts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dashboard", err.Error())
		return
	}

	if diag := SaveDashboardState(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the dashboard.
func (r *DashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DashboardModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if err := r.client.Delete(ctx, data.UID.ValueString(), sdkresource.DeleteOptions{}); err != nil {
		resp.Diagnostics.AddError("Failed to delete dashboard", err.Error())
		return
	}
}

// ImportState imports the state of the dashboard.
func (r *DashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	res, err := r.client.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get dashboard", err.Error())
		return
	}

	var data DashboardModel
	if diag := SaveDashboardState(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the dashboard.
func (r *DashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DashboardModel
	if diag := req.State.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	res, err := r.client.Get(ctx, data.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dashboard", err.Error())
		return
	}

	if diag := SaveDashboardState(ctx, res, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DashboardResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DashboardModel
	if diag := req.Config.Get(ctx, &data); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	var opts DashboardOptions
	if diag := ParseOptions(ctx, data.Options, &opts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if opts.Validate {
		if err := ValidateDashboard([]byte(data.Spec.ValueString())); err != nil {
			resp.Diagnostics.AddError("Invalid dashboard spec", err.Error())
			return
		}
	}

	if len(opts.LintRules) > 0 {
		results, ok, err := LintDashboard(
			GetLintRules(opts.LintRules), []byte(data.Spec.ValueString()),
		)
		if err != nil {
			resp.Diagnostics.AddError("Failed to lint dashboard", err.Error())
			return
		}

		if ok {
			return
		}

		resp.Diagnostics.AddWarning(
			path.Root("spec").String(), results.Warnings,
		)

		resp.Diagnostics.AddError(
			path.Root("spec").String(), results.Errors,
		)
	}
}

func ValidateDashboard(bspec []byte) error {
	var dash dashboard.Dashboard
	return dash.UnmarshalJSONStrict(bspec)
}

var AllRules = []lint.Rule{
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

func GetLintRules(names []string) []lint.Rule {
	res := make([]lint.Rule, 0, len(names))

	for _, name := range names {
		for _, rule := range AllRules {
			if rule.Name() == name {
				res = append(res, rule)
			}
		}
	}

	return res
}

type LintResults struct {
	Warnings string
	Errors   string
}

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

type DashboardNormalizer struct{}

func (n *DashboardNormalizer) Description(context.Context) string {
	return "normalizes the dashboard plan"
}

func (n *DashboardNormalizer) MarkdownDescription(ctx context.Context) string {
	return n.Description(ctx)
}

func (n *DashboardNormalizer) PlanModifyString(
	ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse,
) {
	tflog.Debug(ctx, "normalizing dashboard plan")
}

// ParseDashboard parses a dashboard model into a dashboard resource.
func ParseDashboard(ctx context.Context, src DashboardModel, dst *v0alpha1.Dashboard) diag.Diagnostics {
	tflog.Debug(ctx, "parsing dashboard from model to resource")

	res := make(diag.Diagnostics, 0)

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		res.AddError("Failed to get request dashboard metadata", err.Error())
		return res
	}

	// Set required attributes.
	meta.SetName(src.UID.ValueString())

	// Normally a resource would be constructed from the data model,
	// but for the dashboard we expect the spec to be provided as a stringified JSON.
	if err := json.Unmarshal([]byte(src.Spec.ValueString()), &dst.Spec.Object); err != nil {
		res.AddError("Failed to parse dashboard spec", err.Error())
		return res
	}

	if src.Title.ValueString() != "" {
		dst.Spec.Object["title"] = src.Title.ValueString()
	}

	// Set optional overrides.
	if fid := src.FolderUID.ValueString(); fid != "" {
		meta.SetFolder(fid)
	}

	// Add extra tags, if set.
	if len(src.Tags.Elements()) > 0 {
		tags := make([]types.String, 0, len(src.Tags.Elements()))

		if diag := src.Tags.ElementsAs(ctx, &tags, false); diag.HasError() {
			return diag
		}

		// HACK: because the tags are not a known field in the dashboard spec,
		// we need to manually wrangle them here.
		dashtags := getTags(dst)
		for _, tag := range tags {
			dashtags = append(dashtags, tag.ValueString())
		}

		dst.Spec.Object["tags"] = dashtags
	}

	return res
}

func getTags(src *v0alpha1.Dashboard) []string {
	tags, ok := src.Spec.Object["tags"]
	if !ok {
		return []string{}
	}

	taglist, ok := tags.([]string)
	if !ok {
		return []string{}
	}

	if taglist == nil {
		return []string{}
	}

	return taglist
}

// SaveDashboardState saves the state of a dashboard resource into a dashboard model.
func SaveDashboardState(ctx context.Context, src *v0alpha1.Dashboard, dst *DashboardModel) diag.Diagnostics {
	res := make(diag.Diagnostics, 0)

	meta, err := utils.MetaAccessor(src)
	if err != nil {
		res.AddError("Failed to get response dashboard metadata", err.Error())
		return res
	}

	dst.UUID = types.StringValue(string(meta.GetUID()))
	dst.Version = types.StringValue(meta.GetResourceVersion())
	// TODO: this is a placeholder, we need to construct the URL from the Grafana API.
	dst.URL = types.StringValue(meta.GetSelfLink())

	return res
}

// ParseOptions parses the options for a dashboard model from a Terraform Object type.
func ParseOptions(ctx context.Context, src types.Object, dst *DashboardOptions) diag.Diagnostics {
	if src.IsNull() || src.IsUnknown() {
		return nil
	}

	var opts DashboardModelOptions
	if diag := src.As(ctx, &opts, basetypes.ObjectAsOptions{}); diag.HasError() {
		return diag
	}

	dst.Overwrite = opts.Overwrite.ValueBool()
	dst.Validate = opts.Validate.ValueBool()

	if !opts.LintRules.IsNull() {
		lintRules := make([]string, 0, len(opts.LintRules.Elements()))
		if diag := opts.LintRules.ElementsAs(ctx, &lintRules, false); diag.HasError() {
			return diag
		}

		dst.LintRules = lintRules
	}

	return diag.Diagnostics{}
}
