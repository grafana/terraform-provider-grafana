package k6

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/k6providerapi"
)

type basePluginFrameworkResource struct {
	client *k6.APIClient
	config *k6providerapi.K6APIConfig
}

func (r *basePluginFrameworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
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

	if client.K6APIClient == nil || client.K6APIConfig == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the k6 Cloud API.",
			"Please ensure that k6_access_token and stack_id are set in the provider configuration.",
		)

		return
	}

	r.client = client.K6APIClient
	r.config = client.K6APIConfig
}

type basePluginFrameworkDataSource struct {
	client *k6.APIClient
	config *k6providerapi.K6APIConfig
}

func (d *basePluginFrameworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || d.client != nil {
		return
	}

	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client.K6APIClient == nil || client.K6APIConfig == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the k6 Cloud API.",
			"Please ensure that k6_access_token and stack_id are set in the provider configuration.",
		)

		return
	}

	d.client = client.K6APIClient
	d.config = client.K6APIConfig
}

// getProjectAllowedLoadZones retrieves the allowed load zones for a project
// Returns k6_load_zone_ids directly from the API response
func getProjectAllowedLoadZones(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig, projectID int32) ([]string, error) {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, config.Token)

	resp, _, err := client.LoadZonesAPI.ProjectsAllowedLoadZonesRetrieve(ctx, projectID).
		XStackId(config.StackID).
		Execute()
	if err != nil {
		return nil, err
	}

	var k6LoadZoneIds []string
	for _, zone := range resp.GetValue() {
		k6LoadZoneIds = append(k6LoadZoneIds, zone.GetK6LoadZoneId())
	}

	return k6LoadZoneIds, nil
}

// setProjectAllowedLoadZones updates the allowed load zones for a project
// loadZones parameter contains k6_load_zone_ids, which need to be resolved to actual load zone IDs
func setProjectAllowedLoadZones(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig, projectID int32, k6LoadZoneIds []string) error {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, config.Token)

	// Initialize allowedZones as an empty slice to ensure it's serialized as [] instead of null
	allowedZones := make([]k6.AllowedLoadZoneToUpdateApiModel, 0)

	// Fetch all load zones
	allZonesResp, _, err := client.LoadZonesAPI.LoadZonesList(ctx).
		XStackId(config.StackID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to fetch load zones: %w", err)
	}

	// Track invalid k6_load_zone_ids
	var invalidZoneIds []string

	// Resolve each k6_load_zone_id to actual load zone ID using in-memory matching
	for _, k6LoadZoneID := range k6LoadZoneIds {
		found := false
		for _, zone := range allZonesResp.GetValue() {
			if zone.GetK6LoadZoneId() == k6LoadZoneID {
				// Create an AllowedLoadZoneToUpdateApiModel with the load zone ID
				zoneToUpdate := k6.NewAllowedLoadZoneToUpdateApiModel(zone.GetId())
				allowedZones = append(allowedZones, *zoneToUpdate)
				found = true
				break
			}
		}
		if !found {
			invalidZoneIds = append(invalidZoneIds, k6LoadZoneID)
		}
	}

	// Return error if any invalid zone IDs were found
	if len(invalidZoneIds) > 0 {
		return fmt.Errorf("invalid k6_load_zone_ids: %v", invalidZoneIds)
	}

	updateData := k6.NewUpdateAllowedLoadZonesListApiModel(allowedZones)

	_, httpResp, err := client.LoadZonesAPI.ProjectsAllowedLoadZonesUpdate(ctx, projectID).
		UpdateAllowedLoadZonesListApiModel(updateData).
		XStackId(config.StackID).
		Execute()

	if err != nil && httpResp != nil && httpResp.StatusCode == http.StatusBadRequest {
		// Read the response body to include it in the error message
		if httpResp.Body != nil {
			bodyBytes, readErr := io.ReadAll(httpResp.Body)
			if readErr == nil {
				return fmt.Errorf("API returned 400 Bad Request: %s. Original error: %v", string(bodyBytes), err)
			}
		}
		return fmt.Errorf("API returned 400 Bad Request. Original error: %v", err)
	}

	return err
}
