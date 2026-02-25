package grafana

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/annotations"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &annotationResource{}
	_ resource.ResourceWithConfigure   = &annotationResource{}
	_ resource.ResourceWithImportState = &annotationResource{}

	resourceAnnotationName = "grafana_annotation"
	resourceAnnotationID   = orgResourceIDInt("id")
)

func makeResourceAnnotation() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceAnnotationName,
		resourceAnnotationID,
		&annotationResource{},
	).WithLister(listerFunctionOrgResource(listAnnotationsFramework))
}

type resourceAnnotationModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Text         types.String `tfsdk:"text"`
	Time         types.String `tfsdk:"time"`
	TimeEnd      types.String `tfsdk:"time_end"`
	DashboardUID types.String `tfsdk:"dashboard_uid"`
	PanelID      types.Int64  `tfsdk:"panel_id"`
	Tags         types.Set    `tfsdk:"tags"`
}

type annotationResource struct {
	basePluginFrameworkResource
}

// rfc3339TimeValidator validates that a string is a valid RFC3339 time. Skips validation when the value is null or empty (optional/computed).
type rfc3339TimeValidator struct{}

func (rfc3339TimeValidator) Description(_ context.Context) string {
	return "value must be a valid RFC3339 time string"
}

func (v rfc3339TimeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339TimeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if s == "" {
		return
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, v.Description(ctx), fmt.Sprintf("expected valid RFC3339 date: %s", err.Error()))
	}
}

func (r *annotationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAnnotationName
}

func (r *annotationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Grafana annotations.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/annotate-visualizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/annotations/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"text": schema.StringAttribute{
				Required:    true,
				Description: "The text to associate with the annotation.",
			},
			"time": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The RFC 3339-formatted time string indicating the annotation's time.",
				Validators:  []validator.String{rfc3339TimeValidator{}},
			},
			"time_end": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The RFC 3339-formatted time string indicating the annotation's end time.",
				Validators:  []validator.String{rfc3339TimeValidator{}},
			},
			"dashboard_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The UID of the dashboard on which to create the annotation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"panel_id": schema.Int64Attribute{
				Optional:    true,
				Description: "The ID of the dashboard panel on which to create the annotation.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Description: "The tags to associate with the annotation.",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *annotationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *annotationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceAnnotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	annotation, err := makeAnnotationFromModel(&data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build annotation", err.Error())
		return
	}

	createResp, err := client.Annotations.PostAnnotation(annotation)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create annotation", err.Error())
		return
	}

	// Set ID
	annotationID := *createResp.GetPayload().ID
	data.ID = types.StringValue(MakeOrgResourceID(orgID, annotationID))

	// Read back to get computed values
	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *annotationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceAnnotationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read from API
	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *annotationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceAnnotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, parseErr := r.clientFromExistingOrgResource(resourceAnnotationID, data.ID.ValueString())
	if parseErr != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", parseErr.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no parts")
		return
	}
	idStr := fmt.Sprintf("%v", split[0])

	postAnnotation, err := makeAnnotationFromModel(&data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build annotation", err.Error())
		return
	}

	// Convert to update payload
	annotation := models.UpdateAnnotationsCmd{
		Tags:    postAnnotation.Tags,
		Text:    *postAnnotation.Text,
		Time:    postAnnotation.Time,
		TimeEnd: postAnnotation.TimeEnd,
	}

	_, err = client.Annotations.UpdateAnnotation(idStr, &annotation)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update annotation", err.Error())
		return
	}

	// Read back to get updated values
	readData, diags := r.read(ctx, data.ID.ValueString())
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *annotationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceAnnotationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, parseErr := r.clientFromExistingOrgResource(resourceAnnotationID, data.ID.ValueString())
	if parseErr != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", parseErr.Error())
		return
	}
	if len(split) == 0 {
		resp.Diagnostics.AddError("Invalid resource ID", "Resource ID has no parts")
		return
	}
	idStr := fmt.Sprintf("%v", split[0])

	_, err := client.Annotations.DeleteAnnotationByID(idStr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete annotation", err.Error())
		return
	}
}

func (r *annotationResource) read(ctx context.Context, id string) (*resourceAnnotationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceAnnotationID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	if len(split) == 0 {
		diags.AddError("Invalid resource ID", "Resource ID has no parts")
		return nil, diags
	}
	idStr := fmt.Sprintf("%v", split[0])

	resp, err := client.Annotations.GetAnnotationByID(idStr)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read annotation", err.Error())
		return nil, diags
	}

	annotation := resp.GetPayload()

	// Handle dashboard UID lookup if needed
	if annotation.DashboardID > 0 && annotation.DashboardUID == "" {
		// Have to list annotations here because the dashboard_uid is not fetched when using GetAnnotationByID
		// Also, the GetDashboardByID API is deprecated and removed.
		// TODO: Fix the API. The dashboard UID is not returned in the response.
		listParams := annotations.NewGetAnnotationsParams().
			WithDashboardID(&annotation.DashboardID).
			WithFrom(&annotation.Time).
			WithTo(&annotation.TimeEnd)

		listResp, err := client.Annotations.GetAnnotations(listParams)
		if err != nil {
			diags.AddError("Failed to list annotations", err.Error())
			return nil, diags
		}
		for _, a := range listResp.Payload {
			if strconv.FormatInt(a.ID, 10) == idStr {
				annotation.DashboardUID = a.DashboardUID
				break
			}
		}
	}

	// Convert times from milliseconds to RFC3339
	t := time.UnixMilli(annotation.Time)
	tEnd := time.UnixMilli(annotation.TimeEnd)

	// Convert tags to Framework set; use null when empty so state matches plan (optional attribute unset).
	var tags types.Set
	if len(annotation.Tags) == 0 {
		tags = types.SetNull(types.StringType)
	} else {
		var tagDiags diag.Diagnostics
		tags, tagDiags = types.SetValueFrom(ctx, types.StringType, annotation.Tags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return nil, diags
		}
	}

	dashboardUID := types.StringNull()
	if annotation.DashboardUID != "" {
		dashboardUID = types.StringValue(annotation.DashboardUID)
	}
	panelID := types.Int64Null()
	if annotation.PanelID != 0 {
		panelID = types.Int64Value(annotation.PanelID)
	}

	data := &resourceAnnotationModel{
		ID:           types.StringValue(MakeOrgResourceID(orgID, annotation.ID)),
		OrgID:        types.StringValue(strconv.FormatInt(orgID, 10)),
		Text:         types.StringValue(annotation.Text),
		Time:         types.StringValue(t.Format(time.RFC3339)),
		TimeEnd:      types.StringValue(tEnd.Format(time.RFC3339)),
		DashboardUID: dashboardUID,
		PanelID:      panelID,
		Tags:         tags,
	}

	return data, diags
}

// Helper functions

func listAnnotationsFramework(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Annotations.GetAnnotations(annotations.NewGetAnnotationsParams())
	if err != nil {
		return nil, err
	}

	for _, annotation := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, annotation.ID))
	}

	return ids, nil
}

func makeAnnotationFromModel(data *resourceAnnotationModel) (*models.PostAnnotationsCmd, error) {
	text := data.Text.ValueString()
	a := &models.PostAnnotationsCmd{
		Text:         &text,
		PanelID:      data.PanelID.ValueInt64(),
		DashboardUID: data.DashboardUID.ValueString(),
	}

	// Convert tags from Framework set to string slice
	if !data.Tags.IsNull() {
		var tags []string
		tagDiags := data.Tags.ElementsAs(context.Background(), &tags, false)
		if tagDiags.HasError() {
			return nil, fmt.Errorf("failed to convert tags: %s", tagDiags.Errors()[0].Summary())
		}
		a.Tags = tags
	}

	// Convert time strings to milliseconds since epoch
	if !data.Time.IsNull() && data.Time.ValueString() != "" {
		t, err := millisSinceEpochFromString(data.Time.ValueString())
		if err != nil {
			return nil, err
		}
		a.Time = t
	}

	if !data.TimeEnd.IsNull() && data.TimeEnd.ValueString() != "" {
		tEnd, err := millisSinceEpochFromString(data.TimeEnd.ValueString())
		if err != nil {
			return nil, err
		}
		a.TimeEnd = tEnd
	}

	return a, nil
}

func millisSinceEpochFromString(timeStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / int64(time.Millisecond), nil
}
