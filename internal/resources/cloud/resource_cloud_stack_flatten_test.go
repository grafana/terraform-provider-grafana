package cloud

import (
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitFlattenStack_BasicStackFields(t *testing.T) {
	t.Parallel()

	stack := &gcom.FormattedApiInstance{
		Id:                              12345,
		Name:                            "my-stack",
		Slug:                            "my-stack",
		Url:                             "https://my-stack.grafana.net",
		Status:                          "active",
		RegionSlug:                      "eu",
		ClusterSlug:                     "prod-eu-west-0",
		ClusterName:                     "EU West 0",
		Description:                     "test stack",
		DeleteProtection:                true,
		HmInstancePromId:                101,
		HmInstancePromUrl:               "https://prometheus-prod-04.csp-region-1.grafana.net",
		HmInstancePromName:              "prometheus",
		HmInstancePromStatus:            "active",
		RegionSyntheticMonitoringApiUrl: "https://synthetic-monitoring-api-eu-west-0.grafana.net",
	}

	connections := gcom.NewStackConnectionsV1([]gcom.StackConnectionTenantV1{})

	d := schema.TestResourceDataRaw(t, resourceStack().Schema.Schema, map[string]any{})
	if err := flattenStack(d, stack, connections, nil); err != nil {
		t.Fatalf("flattenStack: %v", err)
	}

	requireStringAttr(t, d, "id", "12345")
	requireStringAttr(t, d, "name", "my-stack")
	requireStringAttr(t, d, "slug", "my-stack")
	requireStringAttr(t, d, "url", "https://my-stack.grafana.net")
	requireStringAttr(t, d, "status", "active")
	requireStringAttr(t, d, "region_slug", "eu")
	requireStringAttr(t, d, "cluster_slug", "prod-eu-west-0")
	requireStringAttr(t, d, "cluster_name", "EU West 0")
	requireStringAttr(t, d, "description", "test stack")
	requireBoolAttr(t, d, "delete_protection", true)
	requireIntAttr(t, d, "prometheus_user_id", 101)
	requireStringAttr(t, d, "prometheus_remote_endpoint", "https://prometheus-prod-04.csp-region-1.grafana.net/api/prom")
	requireStringAttr(t, d, "prometheus_remote_write_endpoint", "https://prometheus-prod-04.csp-region-1.grafana.net/api/prom/push")
	requireStringAttr(t, d, "sm_url", "https://synthetic-monitoring-api-eu-west-0.grafana.net")
	requireStringAttr(t, d, "cloud_provider_url", "https://cloud-provider-api-prod-eu-west-0.csp-region-1.grafana.net")
	requireStringAttr(t, d, "connections_api_url", "https://connections-api-prod-eu-west-0.csp-region-1.grafana.net")
}

func TestUnitFlattenStack_StackConnectionsV1(t *testing.T) {
	t.Parallel()

	stack := &gcom.FormattedApiInstance{
		Id:                99,
		Name:              "conn-stack",
		Slug:              "conn-stack",
		Url:               "https://conn-stack.grafana.net",
		Status:            "active",
		ClusterSlug:       "prod-eu-west-0",
		HmInstancePromUrl: "https://prometheus-prod-01-eu-west-0.grafana.net",
	}

	connections := &gcom.StackConnectionsV1{
		Tenants: []gcom.StackConnectionTenantV1{
			{
				Id:   1,
				Type: "grafana",
			},
			{
				Id:   2,
				Type: "prometheus",
				PrivateConnectivityInfo: &gcom.BasicPrivateConnectivityInfo{
					PrivateDNS:  common.Ref("prom-private.example.net"),
					ServiceName: common.Ref("com.amazonaws.vpce.eu-west-1.vpce-svc-prom"),
					Regions:     []string{"eu-west-1"},
				},
			},
			{
				Id:   3,
				Type: "alerts",
			},
			{
				Id:   4,
				Type: "agent-management",
				PrivateConnectivityInfo: &gcom.BasicPrivateConnectivityInfo{
					PrivateDNS:  common.Ref("fleet-private.example.net"),
					ServiceName: common.Ref("com.amazonaws.vpce.eu-west-1.vpce-svc-fleet"),
				},
			},
		},
		Services: &gcom.StackConnectionServicesV1{
			OncallApiUrl: common.Ref("https://oncall-prod-eu-west-0.grafana.net/oncall"),
			InfluxUrl:    common.Ref("https://influx-prod-eu-west-0.grafana.net"),
		},
		Otlp: &gcom.StackConnectionOtlpV1{
			Url: common.Ref("https://otlp-prod-eu-west-0.grafana.net/otlp"),
			PrivateConnectivityInfo: &gcom.BasicPrivateConnectivityInfo{
				PrivateDNS:  common.Ref("otlp-private.example.net"),
				ServiceName: common.Ref("com.amazonaws.vpce.eu-west-1.vpce-svc-otlp"),
			},
		},
		Pdc: &gcom.StackConnectionPdcV1{
			Api: gcom.BasicPrivateConnectivityInfo{
				PrivateDNS:  common.Ref("pdc-api-private.example.net"),
				ServiceName: common.Ref("com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-api"),
			},
			Gateway: gcom.BasicPrivateConnectivityInfo{
				PrivateDNS:  common.Ref("pdc-gateway-private.example.net"),
				ServiceName: common.Ref("com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-gateway"),
			},
		},
	}

	ipAllowListCNAMByTenantType := map[string]string{
		"grafana":    "grafanas.example.net",
		"prometheus": "prom.example.net",
		"alerts":     "alerts.example.net",
	}

	d := schema.TestResourceDataRaw(t, resourceStack().Schema.Schema, map[string]any{})
	if err := flattenStack(d, stack, connections, ipAllowListCNAMByTenantType); err != nil {
		t.Fatalf("flattenStack: %v", err)
	}

	requireStringAttr(t, d, "grafanas_ip_allow_list_cname", "grafanas.example.net")
	requireStringAttr(t, d, "prometheus_ip_allow_list_cname", "prom.example.net")
	requireStringAttr(t, d, "prometheus_private_connectivity_info_private_dns", "prom-private.example.net")
	requireStringAttr(t, d, "prometheus_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-prom")
	requireStringSliceAttr(t, d, "prometheus_private_connectivity_info_regions", []string{"eu-west-1"})
	requireStringAttr(t, d, "alertmanager_ip_allow_list_cname", "alerts.example.net")
	requireStringAttr(t, d, "fleet_management_private_connectivity_info_private_dns", "fleet-private.example.net")
	requireStringAttr(t, d, "fleet_management_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-fleet")
	requireStringAttr(t, d, "oncall_api_url", "https://oncall-prod-eu-west-0.grafana.net/oncall")
	requireStringAttr(t, d, "influx_url", "https://influx-prod-eu-west-0.grafana.net")
	requireStringAttr(t, d, "otlp_url", "https://otlp-prod-eu-west-0.grafana.net/otlp")
	requireStringAttr(t, d, "otlp_private_connectivity_info_private_dns", "otlp-private.example.net")
	requireStringAttr(t, d, "otlp_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-otlp")
	requireStringAttr(t, d, "pdc_api_private_connectivity_info_private_dns", "pdc-api-private.example.net")
	requireStringAttr(t, d, "pdc_api_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-api")
	requireStringAttr(t, d, "pdc_gateway_private_connectivity_info_private_dns", "pdc-gateway-private.example.net")
	requireStringAttr(t, d, "pdc_gateway_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-gateway")
}

func TestUnitIPAllowListCNAMByTenantType(t *testing.T) {
	t.Parallel()

	tenants := []gcom.TenantsInner{
		{
			Type:             "grafana",
			IpAllowListCNAME: *gcom.NewNullableString(common.Ref("grafanas.example.net")),
		},
		{
			Type:             "prometheus",
			IpAllowListCNAME: *gcom.NewNullableString(common.Ref("prom.example.net")),
		},
		{
			Type: "logs",
		},
	}

	got := ipAllowListCNAMByTenantType(tenants)

	want := map[string]string{
		"grafana":    "grafanas.example.net",
		"prometheus": "prom.example.net",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for tenantType, cname := range want {
		if got[tenantType] != cname {
			t.Fatalf("%s: got %q, want %q", tenantType, got[tenantType], cname)
		}
	}
}

func requireStringAttr(t *testing.T, d *schema.ResourceData, key, want string) {
	t.Helper()

	got, ok := d.GetOk(key)
	if !ok {
		t.Fatalf("%s: attribute not set", key)
	}
	if got.(string) != want {
		t.Fatalf("%s: got %q, want %q", key, got, want)
	}
}

func requireBoolAttr(t *testing.T, d *schema.ResourceData, key string, want bool) {
	t.Helper()

	got, ok := d.GetOk(key)
	if !ok {
		t.Fatalf("%s: attribute not set", key)
	}
	if got.(bool) != want {
		t.Fatalf("%s: got %v, want %v", key, got, want)
	}
}

func requireIntAttr(t *testing.T, d *schema.ResourceData, key string, want int) {
	t.Helper()

	got, ok := d.GetOk(key)
	if !ok {
		t.Fatalf("%s: attribute not set", key)
	}
	if got.(int) != want {
		t.Fatalf("%s: got %v, want %v", key, got, want)
	}
}

func requireStringSliceAttr(t *testing.T, d *schema.ResourceData, key string, want []string) {
	t.Helper()

	got, ok := d.GetOk(key)
	if !ok {
		t.Fatalf("%s: attribute not set", key)
	}

	gotSlice, ok := got.([]string)
	if !ok {
		raw, ok := got.([]any)
		if !ok {
			t.Fatalf("%s: got %T, want []string", key, got)
		}
		gotSlice = make([]string, len(raw))
		for i, v := range raw {
			s, ok := v.(string)
			if !ok {
				t.Fatalf("%s[%d]: got %T, want string", key, i, v)
			}
			gotSlice[i] = s
		}
	}
	if len(gotSlice) != len(want) {
		t.Fatalf("%s: got %v, want %v", key, gotSlice, want)
	}
	for i := range want {
		if gotSlice[i] != want[i] {
			t.Fatalf("%s[%d]: got %q, want %q", key, i, gotSlice[i], want[i])
		}
	}
}
