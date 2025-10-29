package cloud

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceIPs() *common.DataSource {
	schema := &schema.Resource{
		Description: "Data source for retrieving sets of cloud IPs. See https://grafana.com/docs/grafana-cloud/reference/allow-list/ for more info",
		ReadContext: datasourceIPsRead,
		Schema: map[string]*schema.Schema{
			"hosted_alerts": {
				Description: "Set of IP addresses that are used for hosted alerts.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosted_grafana": {
				Description: "Set of IP addresses that are used for hosted Grafana.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosted_metrics": {
				Description: "Set of IP addresses that are used for hosted metrics.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosted_traces": {
				Description: "Set of IP addresses that are used for hosted traces.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosted_logs": {
				Description: "Set of IP addresses that are used for hosted logs.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hosted_profiles": {
				Description: "Set of IP addresses that are used for hosted profiles.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryCloud, "grafana_cloud_ips", schema)
}

func datasourceIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	d.SetId("cloud_ips")
	for attr, dataURL := range map[string]string{
		"hosted_alerts":   "https://grafana.com/api/hosted-alerts/source-ips.txt",
		"hosted_grafana":  "https://grafana.com/api/hosted-grafana/source-ips.txt",
		"hosted_metrics":  "https://grafana.com/api/hosted-metrics/source-ips.txt",
		"hosted_traces":   "https://grafana.com/api/hosted-traces/source-ips.txt",
		"hosted_logs":     "https://grafana.com/api/hosted-logs/source-ips.txt",
		"hosted_profiles": "https://grafana.com/api/hosted-profiles/source-ips.txt",
	} {
		// nolint: gosec
		resp, err := http.Get(dataURL)
		if err != nil {
			return diag.Errorf("error querying IPs for %s (%s): %v", attr, dataURL, err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return diag.Errorf("error reading response body for %s (%s): %v", attr, dataURL, err)
		}
		var ipStr []string
		for _, ip := range strings.Split(string(b), "\n") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				ipStr = append(ipStr, ip)
			}
		}

		if err := d.Set(attr, ipStr); err != nil {
			return diag.Errorf("error setting %s: %v", attr, err)
		}
	}

	return nil
}
