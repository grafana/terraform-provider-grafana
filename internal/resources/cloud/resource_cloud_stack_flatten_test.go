package cloud

import (
	"strconv"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var privateConnectivityPrefixes = []string{
	"prometheus",
	"logs",
	"traces",
	"profiles",
	"graphite",
	"fleet_management",
	"otlp",
	"pdc_api",
	"pdc_gateway",
}

var privateConnectivityListSuffixes = []string{
	"regions",
	"availability_zones",
	"availability_zone_ids",
}

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
	requireEmptyPrivateConnectivityListsInStateForAll(t, d, privateConnectivityPrefixes...)
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
	requireEmptyPrivateConnectivityListsInState(t, d, "prometheus", "availability_zones", "availability_zone_ids")
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
	requireEmptyPrivateConnectivityListsInStateForAll(t, d,
		"fleet_management",
		"otlp",
		"pdc_api",
		"pdc_gateway",
	)
}

func TestUnitFlattenStack_PrivateConnectivityLists(t *testing.T) {
	t.Parallel()

	regions := []string{"eu-west-1", "us-east-1"}
	availabilityZones := []string{"euw1-az1", "use1-az2"}
	availabilityZoneIDs := []string{"euw1-az1-id", "use1-az2-id"}

	privateConnectivityInfo := func(dns, serviceName string) *gcom.BasicPrivateConnectivityInfo {
		return &gcom.BasicPrivateConnectivityInfo{
			PrivateDNS:          common.Ref(dns),
			ServiceName:         common.Ref(serviceName),
			Regions:             regions,
			AvailabilityZones:   availabilityZones,
			AvailabilityZoneIds: availabilityZoneIDs,
		}
	}

	stack := &gcom.FormattedApiInstance{
		Id:                42,
		Name:              "lists-stack",
		Slug:              "lists-stack",
		Url:               "https://lists-stack.grafana.net",
		Status:            "active",
		ClusterSlug:       "prod-eu-west-0",
		HmInstancePromUrl: "https://prometheus-prod-01-eu-west-0.grafana.net",
	}

	connections := &gcom.StackConnectionsV1{
		Tenants: []gcom.StackConnectionTenantV1{
			{
				Id:                      1,
				Type:                    "prometheus",
				PrivateConnectivityInfo: privateConnectivityInfo("prom-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-prom"),
			},
			{
				Id:                      2,
				Type:                    "logs",
				PrivateConnectivityInfo: privateConnectivityInfo("logs-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-logs"),
			},
			{
				Id:                      3,
				Type:                    "traces",
				PrivateConnectivityInfo: privateConnectivityInfo("traces-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-traces"),
			},
			{
				Id:                      4,
				Type:                    "profiles",
				PrivateConnectivityInfo: privateConnectivityInfo("profiles-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-profiles"),
			},
			{
				Id:                      5,
				Type:                    "graphite",
				PrivateConnectivityInfo: privateConnectivityInfo("graphite-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-graphite"),
			},
			{
				Id:                      6,
				Type:                    "agent-management",
				PrivateConnectivityInfo: privateConnectivityInfo("fleet-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-fleet"),
			},
		},
		Otlp: &gcom.StackConnectionOtlpV1{
			Url:                     common.Ref("https://otlp-prod-eu-west-0.grafana.net/otlp"),
			PrivateConnectivityInfo: privateConnectivityInfo("otlp-private.example.net", "com.amazonaws.vpce.eu-west-1.vpce-svc-otlp"),
		},
		Pdc: &gcom.StackConnectionPdcV1{
			Api: *privateConnectivityInfo(
				"pdc-api-private.example.net",
				"com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-api",
			),
			Gateway: *privateConnectivityInfo(
				"pdc-gateway-private.example.net",
				"com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-gateway",
			),
		},
	}

	d := schema.TestResourceDataRaw(t, resourceStack().Schema.Schema, map[string]any{})
	if err := flattenStack(d, stack, connections, nil); err != nil {
		t.Fatalf("flattenStack: %v", err)
	}

	for _, prefix := range privateConnectivityPrefixes {
		t.Run(prefix, func(t *testing.T) {
			requirePrivateConnectivityLists(t, d, prefix, regions, availabilityZones, availabilityZoneIDs)
		})
	}
}

func TestUnitFlattenStack_StackConnectionsV1_MissingAgentManagement(t *testing.T) {
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
	requireEmptyPrivateConnectivityListsInState(t, d, "prometheus", "availability_zones", "availability_zone_ids")
	requireStringAttr(t, d, "alertmanager_ip_allow_list_cname", "alerts.example.net")
	requireAttrNotSet(t, d, "fleet_management_private_connectivity_info_private_dns")
	requireAttrNotSet(t, d, "fleet_management_private_connectivity_info_service_name")
	requireEmptyPrivateConnectivityListsInStateForAll(t, d,
		"logs",
		"traces",
		"profiles",
		"graphite",
		"fleet_management",
	)
	requireStringAttr(t, d, "oncall_api_url", "https://oncall-prod-eu-west-0.grafana.net/oncall")
	requireStringAttr(t, d, "influx_url", "https://influx-prod-eu-west-0.grafana.net")
	requireStringAttr(t, d, "otlp_url", "https://otlp-prod-eu-west-0.grafana.net/otlp")
	requireStringAttr(t, d, "otlp_private_connectivity_info_private_dns", "otlp-private.example.net")
	requireStringAttr(t, d, "otlp_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-otlp")
	requireStringAttr(t, d, "pdc_api_private_connectivity_info_private_dns", "pdc-api-private.example.net")
	requireStringAttr(t, d, "pdc_api_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-api")
	requireStringAttr(t, d, "pdc_gateway_private_connectivity_info_private_dns", "pdc-gateway-private.example.net")
	requireStringAttr(t, d, "pdc_gateway_private_connectivity_info_service_name", "com.amazonaws.vpce.eu-west-1.vpce-svc-pdc-gateway")
	requireEmptyPrivateConnectivityListsInStateForAll(t, d, "otlp", "pdc_api", "pdc_gateway")
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

func requireAttrNotSet(t *testing.T, d *schema.ResourceData, key string) {
	t.Helper()

	got, ok := d.GetOk(key)
	if ok {
		t.Fatalf("%s: attribute set with value %v", key, got)
	}
}

func requireComputedListCountInState(t *testing.T, d *schema.ResourceData, key string, want int) {
	t.Helper()

	state := d.State()
	if state == nil {
		t.Fatalf("%s: resource state is nil", key)
	}

	got, ok := state.Attributes[key+".#"]
	if !ok {
		t.Fatalf("%s.#: not present in state", key)
	}
	if got != strconv.Itoa(want) {
		t.Fatalf("%s.#: got %q, want %q", key, got, strconv.Itoa(want))
	}
}

func requireEmptyPrivateConnectivityListsInStateForAll(t *testing.T, d *schema.ResourceData, prefixes ...string) {
	t.Helper()

	for _, prefix := range prefixes {
		requireEmptyPrivateConnectivityListsInState(t, d, prefix)
	}
}

func requireEmptyPrivateConnectivityListsInState(t *testing.T, d *schema.ResourceData, prefix string, suffixes ...string) {
	t.Helper()

	if len(suffixes) == 0 {
		suffixes = privateConnectivityListSuffixes
	}

	for _, suffix := range suffixes {
		requireComputedListCountInState(t, d, prefix+"_private_connectivity_info_"+suffix, 0)
	}
}

func requirePrivateConnectivityLists(
	t *testing.T,
	d *schema.ResourceData,
	prefix string,
	regions, availabilityZones, availabilityZoneIDs []string,
) {
	t.Helper()

	requireStringSliceAttr(t, d, prefix+"_private_connectivity_info_regions", regions)
	requireStringSliceAttr(t, d, prefix+"_private_connectivity_info_availability_zones", availabilityZones)
	requireStringSliceAttr(t, d, prefix+"_private_connectivity_info_availability_zone_ids", availabilityZoneIDs)
	requireComputedListCountInState(t, d, prefix+"_private_connectivity_info_regions", len(regions))
	requireComputedListCountInState(t, d, prefix+"_private_connectivity_info_availability_zones", len(availabilityZones))
	requireComputedListCountInState(t, d, prefix+"_private_connectivity_info_availability_zone_ids", len(availabilityZoneIDs))
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
