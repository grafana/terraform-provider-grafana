package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// StackModel represents the common model for both resource and data source
type StackModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	RegionSlug  types.String `tfsdk:"region_slug"`
	ClusterSlug types.String `tfsdk:"cluster_slug"`
	ClusterName types.String `tfsdk:"cluster_name"`
	OrgID       types.Int64  `tfsdk:"org_id"`
	OrgSlug     types.String `tfsdk:"org_slug"`
	OrgName     types.String `tfsdk:"org_name"`
	Status      types.String `tfsdk:"status"`
	Labels      types.Map    `tfsdk:"labels"`

	// Delete protection (resource only, will be null for data source)
	DeleteProtection types.Bool `tfsdk:"delete_protection"`

	// Wait for readiness (resource only, will be null for data source)
	WaitForReadiness        types.Bool   `tfsdk:"wait_for_readiness"`
	WaitForReadinessTimeout types.String `tfsdk:"wait_for_readiness_timeout"`

	// IP Allow List CNAMEs
	GrafanaIPAllowListCNAME types.String `tfsdk:"grafanas_ip_allow_list_cname"`

	// Prometheus (Metrics/Mimir)
	PrometheusUserID                             types.Int64  `tfsdk:"prometheus_user_id"`
	PrometheusURL                                types.String `tfsdk:"prometheus_url"`
	PrometheusName                               types.String `tfsdk:"prometheus_name"`
	PrometheusRemoteEndpoint                     types.String `tfsdk:"prometheus_remote_endpoint"`
	PrometheusRemoteWriteEndpoint                types.String `tfsdk:"prometheus_remote_write_endpoint"`
	PrometheusStatus                             types.String `tfsdk:"prometheus_status"`
	PrometheusPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"prometheus_private_connectivity_info_private_dns"`
	PrometheusPrivateConnectivityInfoServiceName types.String `tfsdk:"prometheus_private_connectivity_info_service_name"`
	PrometheusIPAllowListCNAME                   types.String `tfsdk:"prometheus_ip_allow_list_cname"`

	// Alertmanager
	AlertmanagerUserID           types.Int64  `tfsdk:"alertmanager_user_id"`
	AlertmanagerName             types.String `tfsdk:"alertmanager_name"`
	AlertmanagerURL              types.String `tfsdk:"alertmanager_url"`
	AlertmanagerStatus           types.String `tfsdk:"alertmanager_status"`
	AlertmanagerIPAllowListCNAME types.String `tfsdk:"alertmanager_ip_allow_list_cname"`

	// OnCall
	OnCallAPIURL types.String `tfsdk:"oncall_api_url"`

	// Logs (Loki)
	LogsUserID                             types.Int64  `tfsdk:"logs_user_id"`
	LogsName                               types.String `tfsdk:"logs_name"`
	LogsURL                                types.String `tfsdk:"logs_url"`
	LogsStatus                             types.String `tfsdk:"logs_status"`
	LogsPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"logs_private_connectivity_info_private_dns"`
	LogsPrivateConnectivityInfoServiceName types.String `tfsdk:"logs_private_connectivity_info_service_name"`
	LogsIPAllowListCNAME                   types.String `tfsdk:"logs_ip_allow_list_cname"`

	// Traces (Tempo)
	TracesUserID                             types.Int64  `tfsdk:"traces_user_id"`
	TracesName                               types.String `tfsdk:"traces_name"`
	TracesURL                                types.String `tfsdk:"traces_url"`
	TracesStatus                             types.String `tfsdk:"traces_status"`
	TracesPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"traces_private_connectivity_info_private_dns"`
	TracesPrivateConnectivityInfoServiceName types.String `tfsdk:"traces_private_connectivity_info_service_name"`
	TracesIPAllowListCNAME                   types.String `tfsdk:"traces_ip_allow_list_cname"`

	// Profiles (Pyroscope)
	ProfilesUserID                             types.Int64  `tfsdk:"profiles_user_id"`
	ProfilesName                               types.String `tfsdk:"profiles_name"`
	ProfilesURL                                types.String `tfsdk:"profiles_url"`
	ProfilesStatus                             types.String `tfsdk:"profiles_status"`
	ProfilesPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"profiles_private_connectivity_info_private_dns"`
	ProfilesPrivateConnectivityInfoServiceName types.String `tfsdk:"profiles_private_connectivity_info_service_name"`
	ProfilesIPAllowListCNAME                   types.String `tfsdk:"profiles_ip_allow_list_cname"`

	// Graphite
	GraphiteUserID                             types.Int64  `tfsdk:"graphite_user_id"`
	GraphiteName                               types.String `tfsdk:"graphite_name"`
	GraphiteURL                                types.String `tfsdk:"graphite_url"`
	GraphiteStatus                             types.String `tfsdk:"graphite_status"`
	GraphitePrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"graphite_private_connectivity_info_private_dns"`
	GraphitePrivateConnectivityInfoServiceName types.String `tfsdk:"graphite_private_connectivity_info_service_name"`
	GraphiteIPAllowListCNAME                   types.String `tfsdk:"graphite_ip_allow_list_cname"`

	// Fleet Management
	FleetManagementUserID                             types.Int64  `tfsdk:"fleet_management_user_id"`
	FleetManagementName                               types.String `tfsdk:"fleet_management_name"`
	FleetManagementURL                                types.String `tfsdk:"fleet_management_url"`
	FleetManagementStatus                             types.String `tfsdk:"fleet_management_status"`
	FleetManagementPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"fleet_management_private_connectivity_info_private_dns"`
	FleetManagementPrivateConnectivityInfoServiceName types.String `tfsdk:"fleet_management_private_connectivity_info_service_name"`

	// Connections
	InfluxURL                              types.String `tfsdk:"influx_url"`
	OtlpURL                                types.String `tfsdk:"otlp_url"`
	OtlpPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"otlp_private_connectivity_info_private_dns"`
	OtlpPrivateConnectivityInfoServiceName types.String `tfsdk:"otlp_private_connectivity_info_service_name"`

	// PDC
	PdcAPIPrivateConnectivityInfoPrivateDNS      types.String `tfsdk:"pdc_api_private_connectivity_info_private_dns"`
	PdcAPIPrivateConnectivityInfoServiceName     types.String `tfsdk:"pdc_api_private_connectivity_info_service_name"`
	PdcGatewayPrivateConnectivityInfoPrivateDNS  types.String `tfsdk:"pdc_gateway_private_connectivity_info_private_dns"`
	PdcGatewayPrivateConnectivityInfoServiceName types.String `tfsdk:"pdc_gateway_private_connectivity_info_service_name"`
}

// setBasicFields sets the basic fields on the model from the stack
func setBasicFields(model *StackModel, stack *gcom.FormattedApiInstance) {
	id := strconv.FormatInt(int64(stack.Id), 10)
	model.ID = types.StringValue(id)
	model.Name = types.StringValue(stack.Name)
	model.Slug = types.StringValue(stack.Slug)
	model.URL = types.StringValue(stack.Url)
	model.Status = types.StringValue(stack.Status)
	model.RegionSlug = types.StringValue(stack.RegionSlug)
	model.ClusterSlug = types.StringValue(stack.ClusterSlug)
	model.ClusterName = types.StringValue(stack.ClusterName)

	if stack.Description != "" {
		model.Description = types.StringValue(stack.Description)
	} else {
		model.Description = types.StringNull()
	}

	// Labels
	if len(stack.Labels) > 0 {
		labelsMap := make(map[string]string)
		for k, v := range stack.Labels {
			if strVal, ok := v.(string); ok {
				labelsMap[k] = strVal
			}
		}
		labels, _ := types.MapValueFrom(context.Background(), types.StringType, labelsMap)
		model.Labels = labels
	} else {
		model.Labels = types.MapNull(types.StringType)
	}

	model.DeleteProtection = types.BoolValue(stack.DeleteProtection)
}

// setPrometheusFields sets Prometheus-related fields on the model
func setPrometheusFields(model *StackModel, stack *gcom.FormattedApiInstance) error {
	if stack.HmInstancePromId > 0 {
		model.PrometheusUserID = types.Int64Value(int64(stack.HmInstancePromId))
	} else {
		model.PrometheusUserID = types.Int64Null()
	}
	model.PrometheusURL = stringValueOrNull(stack.HmInstancePromUrl)
	model.PrometheusName = stringValueOrNull(stack.HmInstancePromName)

	if stack.HmInstancePromUrl != "" {
		reURL, err := appendPath(stack.HmInstancePromUrl, "/api/prom")
		if err != nil {
			return err
		}
		model.PrometheusRemoteEndpoint = types.StringValue(reURL)

		rweURL, err := appendPath(stack.HmInstancePromUrl, "/api/prom/push")
		if err != nil {
			return err
		}
		model.PrometheusRemoteWriteEndpoint = types.StringValue(rweURL)
	} else {
		model.PrometheusRemoteEndpoint = types.StringNull()
		model.PrometheusRemoteWriteEndpoint = types.StringNull()
	}

	model.PrometheusStatus = stringValueOrNull(stack.HmInstancePromStatus)
	return nil
}

// setServiceFields sets fields for various services (Logs, Traces, Profiles, Graphite, Fleet Management)
func setServiceFields(model *StackModel, stack *gcom.FormattedApiInstance) {
	// Alertmanager
	if stack.AmInstanceId > 0 {
		model.AlertmanagerUserID = types.Int64Value(int64(stack.AmInstanceId))
	} else {
		model.AlertmanagerUserID = types.Int64Null()
	}
	model.AlertmanagerName = stringValueOrNull(stack.AmInstanceName)
	model.AlertmanagerURL = stringValueOrNull(stack.AmInstanceUrl)
	model.AlertmanagerStatus = stringValueOrNull(stack.AmInstanceStatus)

	// Logs
	if stack.HlInstanceId > 0 {
		model.LogsUserID = types.Int64Value(int64(stack.HlInstanceId))
	} else {
		model.LogsUserID = types.Int64Null()
	}
	model.LogsURL = stringValueOrNull(stack.HlInstanceUrl)
	model.LogsName = stringValueOrNull(stack.HlInstanceName)
	model.LogsStatus = stringValueOrNull(stack.HlInstanceStatus)

	// Traces
	if stack.HtInstanceId > 0 {
		model.TracesUserID = types.Int64Value(int64(stack.HtInstanceId))
	} else {
		model.TracesUserID = types.Int64Null()
	}
	model.TracesName = stringValueOrNull(stack.HtInstanceName)
	model.TracesURL = stringValueOrNull(stack.HtInstanceUrl)
	model.TracesStatus = stringValueOrNull(stack.HtInstanceStatus)

	// Profiles
	if stack.HpInstanceId > 0 {
		model.ProfilesUserID = types.Int64Value(int64(stack.HpInstanceId))
	} else {
		model.ProfilesUserID = types.Int64Null()
	}
	model.ProfilesName = stringValueOrNull(stack.HpInstanceName)
	model.ProfilesURL = stringValueOrNull(stack.HpInstanceUrl)
	model.ProfilesStatus = stringValueOrNull(stack.HpInstanceStatus)

	// Graphite
	if stack.HmInstanceGraphiteId > 0 {
		model.GraphiteUserID = types.Int64Value(int64(stack.HmInstanceGraphiteId))
	} else {
		model.GraphiteUserID = types.Int64Null()
	}
	model.GraphiteName = stringValueOrNull(stack.HmInstanceGraphiteName)
	model.GraphiteURL = stringValueOrNull(stack.HmInstanceGraphiteUrl)
	model.GraphiteStatus = stringValueOrNull(stack.HmInstanceGraphiteStatus)

	// Fleet Management
	if stack.AgentManagementInstanceId > 0 {
		model.FleetManagementUserID = types.Int64Value(int64(stack.AgentManagementInstanceId))
	} else {
		model.FleetManagementUserID = types.Int64Null()
	}
	model.FleetManagementName = stringValueOrNull(stack.AgentManagementInstanceName)
	model.FleetManagementURL = stringValueOrNull(stack.AgentManagementInstanceUrl)
	model.FleetManagementStatus = stringValueOrNull(stack.AgentManagementInstanceStatus)
}

// setConnectionFields sets connection-related fields (OnCall, OTLP, Influx)
func setConnectionFields(model *StackModel, connections *gcom.FormattedApiInstanceConnections) {
	// OnCall
	if oncallURL := connections.OncallApiUrl; oncallURL.IsSet() {
		model.OnCallAPIURL = types.StringValue(*oncallURL.Get())
	} else {
		model.OnCallAPIURL = types.StringNull()
	}

	// OTLP
	if otlpURL := connections.OtlpHttpUrl; otlpURL.IsSet() {
		model.OtlpURL = types.StringValue(*otlpURL.Get())
	} else {
		model.OtlpURL = types.StringNull()
	}

	// Influx
	if influxURL := connections.InfluxUrl; influxURL.IsSet() {
		model.InfluxURL = types.StringValue(*influxURL.Get())
	} else {
		model.InfluxURL = types.StringNull()
	}
}

// setPrivateConnectivityFields sets private connectivity and IP allow list fields from tenants
func setPrivateConnectivityFields(model *StackModel, connections *gcom.FormattedApiInstanceConnections) {
	privateConnectivityInfo := connections.PrivateConnectivityInfo
	tenants := privateConnectivityInfo.GetTenants()

	// Process tenants for private connectivity and IP allow lists
	setTenantInfo := func(tenantType string, setPrivate, setIPAllow func(string, string)) {
		for _, tenant := range tenants {
			if tenant.Type == tenantType {
				if tenant.Info != nil && setPrivate != nil {
					setPrivate(tenant.Info.InfoAnyOf.PrivateDNS, tenant.Info.InfoAnyOf.ServiceName)
				}
				if tenant.IpAllowListCNAME != nil && setIPAllow != nil {
					setIPAllow(*tenant.IpAllowListCNAME, "")
				}
				break
			}
		}
	}

	// Grafana
	setTenantInfo("grafana", nil, func(cname, _ string) {
		model.GrafanaIPAllowListCNAME = types.StringValue(cname)
	})

	// Prometheus
	setTenantInfo("prometheus", func(privateDNS, serviceName string) {
		model.PrometheusPrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.PrometheusPrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, func(cname, _ string) {
		model.PrometheusIPAllowListCNAME = types.StringValue(cname)
	})

	// Logs
	setTenantInfo("logs", func(privateDNS, serviceName string) {
		model.LogsPrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.LogsPrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, func(cname, _ string) {
		model.LogsIPAllowListCNAME = types.StringValue(cname)
	})

	// Alertmanager (tenant type is "alerts")
	setTenantInfo("alerts", nil, func(cname, _ string) {
		model.AlertmanagerIPAllowListCNAME = types.StringValue(cname)
	})

	// Traces
	setTenantInfo("traces", func(privateDNS, serviceName string) {
		model.TracesPrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.TracesPrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, func(cname, _ string) {
		model.TracesIPAllowListCNAME = types.StringValue(cname)
	})

	// Profiles
	setTenantInfo("profiles", func(privateDNS, serviceName string) {
		model.ProfilesPrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.ProfilesPrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, func(cname, _ string) {
		model.ProfilesIPAllowListCNAME = types.StringValue(cname)
	})

	// Graphite
	setTenantInfo("graphite", func(privateDNS, serviceName string) {
		model.GraphitePrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.GraphitePrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, func(cname, _ string) {
		model.GraphiteIPAllowListCNAME = types.StringValue(cname)
	})

	// Fleet Management (tenant type is "agent-management")
	setTenantInfo("agent-management", func(privateDNS, serviceName string) {
		model.FleetManagementPrivateConnectivityInfoPrivateDNS = types.StringValue(privateDNS)
		model.FleetManagementPrivateConnectivityInfoServiceName = types.StringValue(serviceName)
	}, nil)

	// OTLP private connectivity
	if privateConnectivityInfo.Otlp != nil && privateConnectivityInfo.Otlp.InfoAnyOf != nil {
		otlp := privateConnectivityInfo.Otlp
		model.OtlpPrivateConnectivityInfoPrivateDNS = types.StringValue(otlp.InfoAnyOf.PrivateDNS)
		model.OtlpPrivateConnectivityInfoServiceName = types.StringValue(otlp.InfoAnyOf.ServiceName)
	} else {
		model.OtlpPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.OtlpPrivateConnectivityInfoServiceName = types.StringNull()
	}

	// PDC
	if privateConnectivityInfo.Pdc != nil {
		pdc := privateConnectivityInfo.Pdc
		model.PdcAPIPrivateConnectivityInfoPrivateDNS = types.StringValue(pdc.Api.InfoAnyOf.PrivateDNS)
		model.PdcAPIPrivateConnectivityInfoServiceName = types.StringValue(pdc.Api.InfoAnyOf.ServiceName)
		model.PdcGatewayPrivateConnectivityInfoPrivateDNS = types.StringValue(pdc.Gateway.InfoAnyOf.PrivateDNS)
		model.PdcGatewayPrivateConnectivityInfoServiceName = types.StringValue(pdc.Gateway.InfoAnyOf.ServiceName)
	} else {
		model.PdcAPIPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.PdcAPIPrivateConnectivityInfoServiceName = types.StringNull()
		model.PdcGatewayPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.PdcGatewayPrivateConnectivityInfoServiceName = types.StringNull()
	}

	// Set null values for IP allow lists that weren't found
	setNullIfUnset := func(field types.String) types.String {
		if field.IsNull() {
			return types.StringNull()
		}
		return field
	}

	model.GrafanaIPAllowListCNAME = setNullIfUnset(model.GrafanaIPAllowListCNAME)
	model.PrometheusIPAllowListCNAME = setNullIfUnset(model.PrometheusIPAllowListCNAME)
	model.LogsIPAllowListCNAME = setNullIfUnset(model.LogsIPAllowListCNAME)
	model.AlertmanagerIPAllowListCNAME = setNullIfUnset(model.AlertmanagerIPAllowListCNAME)
	model.TracesIPAllowListCNAME = setNullIfUnset(model.TracesIPAllowListCNAME)
	model.ProfilesIPAllowListCNAME = setNullIfUnset(model.ProfilesIPAllowListCNAME)
	model.GraphiteIPAllowListCNAME = setNullIfUnset(model.GraphiteIPAllowListCNAME)

	// Set null values for private connectivity that weren't found
	if model.PrometheusPrivateConnectivityInfoPrivateDNS.IsNull() {
		model.PrometheusPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.PrometheusPrivateConnectivityInfoServiceName = types.StringNull()
	}
	if model.LogsPrivateConnectivityInfoPrivateDNS.IsNull() {
		model.LogsPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.LogsPrivateConnectivityInfoServiceName = types.StringNull()
	}
	if model.TracesPrivateConnectivityInfoPrivateDNS.IsNull() {
		model.TracesPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.TracesPrivateConnectivityInfoServiceName = types.StringNull()
	}
	if model.ProfilesPrivateConnectivityInfoPrivateDNS.IsNull() {
		model.ProfilesPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.ProfilesPrivateConnectivityInfoServiceName = types.StringNull()
	}
	if model.GraphitePrivateConnectivityInfoPrivateDNS.IsNull() {
		model.GraphitePrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.GraphitePrivateConnectivityInfoServiceName = types.StringNull()
	}
	if model.FleetManagementPrivateConnectivityInfoPrivateDNS.IsNull() {
		model.FleetManagementPrivateConnectivityInfoPrivateDNS = types.StringNull()
		model.FleetManagementPrivateConnectivityInfoServiceName = types.StringNull()
	}
}

// ReadStackData reads stack data from the API and populates the model
func ReadStackData(ctx context.Context, client *gcom.APIClient, stackSlugOrID string) (*StackModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Get stack instance
	req := client.InstancesAPI.GetInstance(ctx, stackSlugOrID)
	stack, httpResp, err := req.Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			diags.AddError("Stack not found", fmt.Sprintf("Stack %s not found", stackSlugOrID))
			return nil, diags
		}
		diags.AddError("Failed to read stack", err.Error())
		return nil, diags
	}

	if stack.Status == "deleted" {
		diags.AddError("Stack deleted", fmt.Sprintf("Stack %s is deleted", stackSlugOrID))
		return nil, diags
	}

	// Get connections with retry
	var connections *gcom.FormattedApiInstanceConnections
	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, httpResp, err := client.InstancesAPI.GetConnections(ctx, stackSlugOrID).Execute()
		if err != nil {
			if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		connections = resp
		return nil
	})
	if err != nil {
		diags.AddError("Failed to get connections", err.Error())
		return nil, diags
	}

	// Flatten to model
	model, err := flattenStackToModel(stack, connections)
	if err != nil {
		diags.AddError("Failed to flatten stack data", err.Error())
		return nil, diags
	}

	return model, diags
}

// flattenStackToModel converts API response to StackModel
func flattenStackToModel(stack *gcom.FormattedApiInstance, connections *gcom.FormattedApiInstanceConnections) (*StackModel, error) {
	model := &StackModel{}

	// Set basic fields
	setBasicFields(model, stack)

	// Set organization fields
	model.OrgID = types.Int64Value(int64(stack.OrgId))
	model.OrgSlug = types.StringValue(stack.OrgSlug)
	model.OrgName = types.StringValue(stack.OrgName)

	// Set Prometheus fields
	if err := setPrometheusFields(model, stack); err != nil {
		return nil, err
	}

	// Set other service fields
	setServiceFields(model, stack)

	// Set connection fields
	setConnectionFields(model, connections)

	// Set private connectivity and IP allow list fields
	setPrivateConnectivityFields(model, connections)

	return model, nil
}

// stringValueOrNull converts a string to types.String, returning null if empty
func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// appendPath is kept from the original file for URL path operations
func appendPath(baseURL, path string) (string, error) {
	bu, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u, err := bu.Parse(path)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
