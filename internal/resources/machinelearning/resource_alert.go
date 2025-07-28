package machinelearning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/prometheus/common/model"
)

var (
	resourceAlertID   = common.NewResourceID(common.StringIDField("id"))
	resourceAlertName = "grafana_machine_learning_alert"

	// Check interface
	_ resource.ResourceWithImportState = (*alertResource)(nil)
)

func resourceAlert() *common.Resource {
	return common.NewResource(
		common.CategoryMachineLearning,
		resourceAlertName,
		resourceAlertID,
		&alertResource{},
	)
}

type resourceAlertModel struct {
	ID               types.String `tfsdk:"id"`
	JobID            types.String `tfsdk:"job_id"`
	OutlierID        types.String `tfsdk:"outlier_id"`
	Title            types.String `tfsdk:"title"`
	AnomalyCondition types.String `tfsdk:"anomaly_condition"`
	For              types.String `tfsdk:"for"`
	Threshold        types.String `tfsdk:"threshold"`
	Window           types.String `tfsdk:"window"`
	Labels           types.Map    `tfsdk:"labels"`
	Annotations      types.Map    `tfsdk:"annotations"`
	NoDataState      types.String `tfsdk:"no_data_state"`
}

type alertResource struct {
	mlapi *mlapi.Client
}

func (r *alertResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.mlapi != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client.MLAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the MLAPI API.",
			"Please ensure that url and auth are set in the provider configuration.",
		)

		return
	}

	r.mlapi = client.MLAPI
}

func (r *alertResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_machine_learning_alert"
}

func (r *alertResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"job_id": schema.StringAttribute{
				Description: "The forecast this alert belongs to.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("job_id"),
						path.MatchRelative().AtParent().AtName("outlier_id"),
					),
				},
			},
			"outlier_id": schema.StringAttribute{
				Description: "The forecast this alert belongs to.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("job_id"),
						path.MatchRelative().AtParent().AtName("outlier_id"),
					),
				},
			},
			"id": schema.StringAttribute{
				Description: "The ID of the alert.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title of the alert.",
				Required:    true,
			},
			"anomaly_condition": schema.StringAttribute{
				Description: "The condition for when to consider a point as anomalous.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("any", "low", "high"),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("outlier_id")),
				},
			},
			"for": schema.StringAttribute{
				Description: "How long values must be anomalous before firing an alert.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0s"),
				Validators: []validator.String{
					anyDuration(),
				},
			},
			"threshold": schema.StringAttribute{
				Description: "The threshold of points over the window that need to be anomalous to alert.",
				Optional:    true,
			},
			"window": schema.StringAttribute{
				Description: "How much time to average values over",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0s"),
				Validators: []validator.String{
					maxDuration(24 * time.Hour),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels to add to the alert generated in Grafana.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"annotations": schema.MapAttribute{
				Description: "Annotations to add to the alert generated in Grafana.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"no_data_state": schema.StringAttribute{
				Description: "How the alert should be processed when no data is returned by the underlying series",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Alerting", "NoData", "OK"),
				},
			},
		},
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID looks like: /(jobs|outliers)/<jobID>/alerts/<alertID>
	id := strings.TrimLeft(req.ID, "/")
	parts := strings.Split(id, "/")
	if len(parts) != 4 ||
		(parts[0] != "jobs" && parts[0] != "outliers") ||
		parts[2] != "alerts" {
		resp.Diagnostics.AddError("Invalid import ID format", "Import ID must be in the format '/(jobs|outliers)/<jobID>/alerts/<alertID>'")
		return
	}
	model := resourceAlertModel{
		ID: types.StringValue(parts[3]),
	}
	if parts[0] == "jobs" {
		model.JobID = types.StringValue(parts[1])
	} else {
		model.OutlierID = types.StringValue(parts[1])
	}

	data, diags := r.read(ctx, model)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.mlapi == nil {
		resp.Diagnostics.AddError("Client not configured", "Client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceAlertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := alertFromModel(data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to make alert structure", err.Error())
		return
	}
	if data.JobID.ValueString() != "" {
		alert, err = r.mlapi.NewJobAlert(ctx, data.JobID.ValueString(), alert)
	} else {
		alert, err = r.mlapi.NewOutlierAlert(ctx, data.OutlierID.ValueString(), alert)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to create resource", err.Error())
		return
	}

	// Read created resource
	data.ID = types.StringValue(alert.ID)
	readData, diags := r.read(ctx, data)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Unable to read created resource", "Resource not found")
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceAlertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.read(ctx, data)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.mlapi == nil {
		resp.Diagnostics.AddError("Client not configured", "Client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceAlertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert, err := alertFromModel(data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to make alert structure", err.Error())
		return
	}
	if data.JobID.ValueString() != "" {
		_, err = r.mlapi.UpdateJobAlert(ctx, data.JobID.ValueString(), alert)
	} else {
		_, err = r.mlapi.UpdateOutlierAlert(ctx, data.OutlierID.ValueString(), alert)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Resource", err.Error())
		return
	}

	// Read updated resource
	readData, diags := r.read(ctx, data)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Unable to read updated resource", "Resource not found")
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.mlapi == nil {
		resp.Diagnostics.AddError("Client not configured", "Client not configured")
		return
	}

	// Read Terraform plan data into the model
	var data resourceAlertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var err error
	if data.JobID.ValueString() != "" {
		err = r.mlapi.DeleteJobAlert(ctx, data.JobID.ValueString(), data.ID.ValueString())
	} else {
		err = r.mlapi.DeleteOutlierAlert(ctx, data.OutlierID.ValueString(), data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Resource", err.Error())
	}
}

func (r *alertResource) read(ctx context.Context, model resourceAlertModel) (*resourceAlertModel, diag.Diagnostics) {
	if r.mlapi == nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("client not configured", "client not configured")}
	}

	var (
		alert mlapi.Alert
		err   error
	)
	if model.JobID.ValueString() != "" {
		alert, err = r.mlapi.JobAlert(ctx, model.JobID.ValueString(), model.ID.ValueString())
	} else {
		alert, err = r.mlapi.OutlierAlert(ctx, model.OutlierID.ValueString(), model.ID.ValueString())
	}
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to read resource", err.Error())}
	}

	data := &resourceAlertModel{}
	data.ID = model.ID
	data.JobID = model.JobID
	data.OutlierID = model.OutlierID
	data.Title = types.StringValue(alert.Title)
	if alert.AnomalyCondition != "" {
		data.AnomalyCondition = types.StringValue(string(alert.AnomalyCondition))
	}
	data.For = types.StringValue(alert.For.String())
	if alert.Threshold != "" {
		data.Threshold = types.StringValue(alert.Threshold)
	}
	data.Window = types.StringValue(alert.Window.String())
	data.Labels = labelsToMapValue(alert.Labels)
	data.Annotations = labelsToMapValue(alert.Annotations)
	if alert.NoDataState != "" {
		data.NoDataState = types.StringValue(string(alert.NoDataState))
	}

	return data, nil
}

func alertFromModel(model resourceAlertModel) (mlapi.Alert, error) {
	forClause, err := parseDuration(model.For.ValueString())
	if err != nil {
		return mlapi.Alert{}, err
	}
	window, err := parseDuration(model.Window.ValueString())
	if err != nil {
		return mlapi.Alert{}, err
	}
	labels, err := mapToLabels(model.Labels)
	if err != nil {
		return mlapi.Alert{}, err
	}
	annotations, err := mapToLabels(model.Annotations)
	if err != nil {
		return mlapi.Alert{}, err
	}
	return mlapi.Alert{
		ID:               model.ID.ValueString(),
		Title:            model.Title.ValueString(),
		AnomalyCondition: mlapi.AnomalyCondition(model.AnomalyCondition.ValueString()),
		For:              forClause,
		Threshold:        model.Threshold.ValueString(),
		Window:           window,
		Labels:           labels,
		Annotations:      annotations,
		NoDataState:      mlapi.NoDataState(model.NoDataState.ValueString()),
	}, nil
}

func labelsToMapValue(labels map[string]string) basetypes.MapValue {
	if labels == nil {
		return basetypes.NewMapNull(types.StringType)
	}
	values := map[string]attr.Value{}
	for k, v := range labels {
		values[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, values)
}

func mapToLabels(m basetypes.MapValue) (map[string]string, error) {
	if m.IsNull() {
		return nil, nil
	}
	labels := map[string]string{}
	for k, v := range m.Elements() {
		if vString, ok := v.(types.String); ok {
			labels[k] = vString.ValueString()
		} else {
			return nil, fmt.Errorf("invalid label value for %s: %v", k, v)
		}
	}
	return labels, nil
}

func parseDuration(s string) (model.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return model.ParseDuration(s)
}

type durationValidator struct {
	max model.Duration
}

func (v durationValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v durationValidator) MarkdownDescription(_ context.Context) string {
	if v.max == 0 {
		return "value must be a duration like 5m"
	}
	return fmt.Sprintf("value must be a duration less than: %s", v.max)
}

func (v durationValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue

	duration, err := model.ParseDuration(request.ConfigValue.ValueString())

	if err != nil || (v.max > 0 && duration > v.max) {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			request.Path,
			v.Description(ctx),
			value.String(),
		))
	}
}

func anyDuration() validator.String {
	return durationValidator{}
}

func maxDuration(max time.Duration) validator.String {
	return durationValidator{
		max: model.Duration(max),
	}
}
