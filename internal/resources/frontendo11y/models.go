package frontendo11y

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

type FrontendO11yAppTFModel struct {
	ID                 types.String `tfsdk:"id"`
	StackID            types.Int64  `tfsdk:"stack_id"`
	Name               types.String `tfsdk:"name"`
	AllowedOrigins     types.List   `tfsdk:"allowed_origins"`
	ExtraLogAtrributes types.Map    `tfsdk:"extra_log_attributes"`
	Settings           types.Map    `tfsdk:"settings"`
	CollectorEndpoint  types.String `tfsdk:"collector_endpoint"`
}

// toClientModel converts a FrontendO11yAppTFModel instance to a frontendo11yapi.App instance.
// A special converter is needed because the TFModel uses special Terraform types that build upon their underlying Go types for
// supporting Terraform's state management/dependency analysis of the resource and its data.
func (tfData FrontendO11yAppTFModel) toClientModel(ctx context.Context) (frontendo11yapi.App, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}

	var originUrls []types.String
	diags := tfData.AllowedOrigins.ElementsAs(ctx, &originUrls, false)
	conversionDiags.Append(diags...)

	allowedOrigins := []frontendo11yapi.AllowedOrigin{}
	for _, url := range originUrls {
		allowedOrigins = append(allowedOrigins, frontendo11yapi.AllowedOrigin{
			URL: url.ValueString(),
		})
	}

	extraLogAttrs := make(map[string]types.String, len(tfData.ExtraLogAtrributes.Elements()))
	diags = tfData.ExtraLogAtrributes.ElementsAs(ctx, &extraLogAttrs, false)
	conversionDiags.Append(diags...)

	extraLogLabels := []frontendo11yapi.LogLabel{}
	for k, tfv := range extraLogAttrs {
		extraLogLabels = append(extraLogLabels, frontendo11yapi.LogLabel{
			Label: k,
			Value: tfv.ValueString(),
		})
	}

	settings := make(map[string]types.String, len(tfData.Settings.Elements()))
	diags = tfData.Settings.ElementsAs(ctx, &settings, false)
	conversionDiags.Append(diags...)

	actualSettings := make(map[string]string, len(settings))
	for k, tfv := range settings {
		actualSettings[k] = tfv.ValueString()
	}

	appID, diags := parseAppID(tfData.ID)
	conversionDiags.Append(diags...)

	return frontendo11yapi.App{
		ID:                 appID,
		Name:               tfData.Name.ValueString(),
		CORSAllowedOrigins: allowedOrigins,
		ExtraLogLabels:     extraLogLabels,
		Settings:           actualSettings,
		CollectEndpointURL: tfData.CollectorEndpoint.ValueString(),
	}, conversionDiags
}

// convertClientModelToTFModel converts a frontendo11yapi.App instance to a FrontendO11yAppTFModel instance.
// A special converter is needed because the TFModel uses special Terraform types that build upon their underlying Go types for
// supporting Terraform's state management/dependency analysis of the resource and its data.
func convertClientModelToTFModel(stackID int64, app frontendo11yapi.App) (FrontendO11yAppTFModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}

	allowedOrigins := []attr.Value{}
	for _, o := range app.CORSAllowedOrigins {
		allowedOrigins = append(allowedOrigins, types.StringValue(o.URL))
	}
	tfAllowedOriginsValue, diags := types.ListValue(types.StringType, allowedOrigins)
	conversionDiags.Append(diags...)

	extraLogLabels := make(map[string]attr.Value, len(app.ExtraLogLabels))
	for _, label := range app.ExtraLogLabels {
		extraLogLabels[label.Label] = types.StringValue(label.Value)
	}
	tfExtraLogAttributes, diags := types.MapValue(types.StringType, extraLogLabels)
	conversionDiags.Append(diags...)

	settings := make(map[string]attr.Value, len(app.Settings))
	for sk, sv := range app.Settings {
		settings[sk] = types.StringValue(sv)
	}
	tfSettings, diags := types.MapValue(types.StringType, settings)
	conversionDiags.Append(diags...)

	resp := FrontendO11yAppTFModel{
		ID:                 types.StringValue(strconv.FormatInt(app.ID, 10)),
		StackID:            types.Int64Value(stackID),
		Name:               types.StringValue(app.Name),
		AllowedOrigins:     tfAllowedOriginsValue,
		ExtraLogAtrributes: tfExtraLogAttributes,
		Settings:           tfSettings,
		CollectorEndpoint:  types.StringValue(fmt.Sprintf("%s/%s", app.CollectEndpointURL, app.Key)),
	}

	return resp, conversionDiags
}

// parseAppID converts the string ID of the Terraform model to the numeric ID of the
// Frontend Observability API. An unknown, null or empty ID (for example in the plan
// of a new app) becomes 0, which the API ignores on create.
func parseAppID(id types.String) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	if id.IsNull() || id.IsUnknown() || id.ValueString() == "" {
		return 0, diags
	}
	appID, err := strconv.ParseInt(id.ValueString(), 10, 64)
	if err != nil {
		diags.AddError("invalid frontend o11y app ID", fmt.Sprintf("could not parse app ID %q: %s", id.ValueString(), err))
	}
	return appID, diags
}
