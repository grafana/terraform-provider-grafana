package cloudintegrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi/models"
)

var (
	_ resource.ResourceWithConfigure   = (*cloudIntegrationResource)(nil)
	_ resource.ResourceWithImportState = (*cloudIntegrationResource)(nil)
)

var (
	resourceCloudIntegrationName = "grafana_cloud_integration"
	resourceCloudIntegrationID   = common.NewResourceID(common.StringIDField("slug"))
)

func resourceCloudIntegration() *common.Resource {
	return common.NewResource(
		common.CategoryCloudIntegrations,
		resourceCloudIntegrationName,
		resourceCloudIntegrationID,
		&cloudIntegrationResource{},
	)
}

type cloudIntegrationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	InstalledVersion types.String `tfsdk:"installed_version"`
	LatestVersion    types.String `tfsdk:"latest_version"`
	Name             types.String `tfsdk:"name"`
	DashboardFolder  types.String `tfsdk:"dashboard_folder"`
	AlertsEnabled    types.Bool   `tfsdk:"alerts_enabled"`
}

type cloudIntegrationResource struct {
	client       *cloudintegrationsapi.Client
	commonClient *common.Client
}

func (r *cloudIntegrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceCloudIntegrationName
}

func (r *cloudIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Manages Grafana Cloud integrations.

* [Official documentation](https://grafana.com/docs/grafana-cloud/data-configuration/integrations/)

This provider lets you manage Grafana Cloud Integrations.
Alerts can optionally be disabled.

Please note: Grafana Cloud Integrations do not support in-place upgrades, and require a teardown and reapply to resolve version drift.
As such it is recommended to have a separate TF plan for integrations to cleanly destroy them as needed.

Update, only triggered on config change, is implemented as a complete uninstall, then reinstall of the integration in question.

Required access policy scopes:

* folders:read
* folders:write
* dashboards:read
* dashboards:write
* rules:read
* rules:write

Based on: https://grafana.com/docs/grafana/latest/alerting/alerting-rules/alerting-migration/#import-rules-with-grafana-alerting

**Note:** This resource creates folders and dashboards as part of the integration installation process, which requires additional permissions beyond the basic integration scopes.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform resource ID. Set to the integration slug.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the integration to install (e.g., 'docker', 'linux-node').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"installed_version": schema.StringAttribute{
				Description: "The version of the installed integration.",
				Computed:    true,
			},
			"latest_version": schema.StringAttribute{
				Description: "The latest version available for this integration.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The display name of the integration.",
				Computed:    true,
			},
			"dashboard_folder": schema.StringAttribute{
				Description: "The dashboard folder associated with this integration.",
				Computed:    true,
			},
			"alerts_enabled": schema.BoolAttribute{
				Description: "Whether alerts are enabled for this integration.",
				Computed:    true,
				Optional:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *cloudIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if client.CloudIntegrationsAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Cloud Integrations API.",
			"Ensure that url and auth are set in the provider configuration.",
		)
		return
	}

	r.client = client.CloudIntegrationsAPIClient
	r.commonClient = client
}

func (r *cloudIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := plan.Slug.ValueString()

	installed, err := r.client.IsIntegrationInstalled(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check integration status", err.Error())
		return
	}

	if !installed {
		config := toAPIConfig(plan.AlertsEnabled)
		var installErr error
		r.commonClient.WithFolderLock(func() {
			r.commonClient.WithDashboardLock(func() {
				installErr = r.client.InstallIntegration(ctx, slug, config)
			})
		})
		if installErr != nil {
			resp.Diagnostics.AddError("Failed to install integration", installErr.Error())
			return
		}
	}

	integration, err := r.client.GetIntegration(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read integration after install", err.Error())
		return
	}

	plan.ID = plan.Slug
	setModelFromAPI(&plan, integration)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := state.Slug.ValueString()

	integration, err := r.client.GetIntegration(ctx, slug)
	if err != nil {
		if errors.Is(err, cloudintegrationsapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read integration", err.Error())
		return
	}

	if integration.Data.Installation == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = state.Slug
	setModelFromAPI(&state, integration)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := plan.Slug.ValueString()
	plan.ID = plan.Slug

	var uninstallErr error
	r.commonClient.WithFolderLock(func() {
		r.commonClient.WithDashboardLock(func() {
			uninstallErr = r.client.UninstallIntegration(ctx, slug)
		})
	})
	if uninstallErr != nil {
		if errors.Is(uninstallErr, cloudintegrationsapi.ErrNotFound) {
			diags = resp.State.Set(ctx, plan)
			resp.Diagnostics.Append(diags...)
			return
		}
		resp.Diagnostics.AddError("Failed to uninstall integration for update", uninstallErr.Error())
		return
	}

	config := toAPIConfig(plan.AlertsEnabled)
	var installErr error
	r.commonClient.WithFolderLock(func() {
		r.commonClient.WithDashboardLock(func() {
			installErr = r.client.InstallIntegration(ctx, slug, config)
		})
	})
	if installErr != nil {
		resp.Diagnostics.AddError("Failed to install integration. Orphaned alerts and dashboards may be present.", installErr.Error())
		return
	}

	integration, err := r.client.GetIntegration(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read integration after update", err.Error())
		return
	}

	setModelFromAPI(&plan, integration)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var uninstallErr error
	r.commonClient.WithFolderLock(func() {
		r.commonClient.WithDashboardLock(func() {
			uninstallErr = r.client.UninstallIntegration(ctx, state.Slug.ValueString())
		})
	})
	if uninstallErr != nil && !errors.Is(uninstallErr, cloudintegrationsapi.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to uninstall integration", uninstallErr.Error())
	}
}

func (r *cloudIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

func toAPIConfig(alertsEnabled types.Bool) *models.InstallationConfig {
	return &models.InstallationConfig{
		ConfigurableAlerts: &models.ConfigurableAlerts{
			// Note, while API exposes logsDisabled, this is not truly acted upon by the backend.
			// As such, we don't expose this to users via Terraform
			AlertsDisabled: !alertsEnabled.ValueBool(),
		},
	}
}

func setModelFromAPI(model *cloudIntegrationResourceModel, integration *models.GetIntegrationResponse) {
	model.Slug = types.StringValue(integration.Data.Slug)
	model.Name = types.StringValue(integration.Data.Name)
	model.LatestVersion = types.StringValue(integration.Data.Version)
	model.DashboardFolder = types.StringValue(integration.Data.DashboardFolder)

	if integration.Data.Installation != nil {
		model.InstalledVersion = types.StringValue(integration.Data.Installation.Version)
	}

	alertsEnabled := true
	if integration.Data.Installation != nil &&
		integration.Data.Installation.Configuration != nil &&
		integration.Data.Installation.Configuration.ConfigurableAlerts != nil {
		alertsEnabled = !integration.Data.Installation.Configuration.ConfigurableAlerts.AlertsDisabled
	}
	model.AlertsEnabled = types.BoolValue(alertsEnabled)
}
