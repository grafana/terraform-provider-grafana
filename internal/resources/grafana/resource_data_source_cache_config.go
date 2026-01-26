package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func resourceDataSourceCacheConfig() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages cache configuration for a data source (Grafana Enterprise).

Use this resource to enable or disable caching for a particular data source. You can also tune the TTL settings for the cache behaviour, or choose to use defaults.

Deleting this resource will cause the cache to be disabled for the target data source.
`,

		CreateContext: CreateOrUpdateDataSourceCacheConfig,
		UpdateContext: CreateOrUpdateDataSourceCacheConfig,
		ReadContext:   ReadDataSourceCacheConfig,
		DeleteContext: DeleteDataSourceCacheConfig,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"datasource_uid": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "UID of the data source to configure.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether caching is enabled for this data source.",
			},
			"use_default_ttl": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If true, use Grafana's default TTLs instead of custom values.",
			},
			"ttl_queries_ms": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "TTL for query caching, in milliseconds. Ignored if use_default_ttl is true.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diffs when using default TTLs
					if v, ok := d.GetOk("use_default_ttl"); ok && v.(bool) {
						return true
					}
					return false
				},
			},
			"ttl_resources_ms": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "TTL for resource caching, in milliseconds. Ignored if use_default_ttl is true.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diffs when using default TTLs
					if v, ok := d.GetOk("use_default_ttl"); ok && v.(bool) {
						return true
					}
					return false
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_data_source_cache_config",
		orgResourceIDString("datasource_uid"),
		schema,
	)
}

func CreateOrUpdateDataSourceCacheConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	dsUID := d.Get("datasource_uid").(string)

	// If enabled is explicitly set to false, disable via dedicated endpoint to avoid omitempty issues.
	if _, present := d.GetOk("enabled"); present || d.HasChange("enabled") {
		if !d.Get("enabled").(bool) {
			if _, err := client.Enterprise.DisableDataSourceCache(dsUID); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	body := &models.CacheConfigSetter{
		DataSourceUID: dsUID,
	}
	if _, present := d.GetOk("enabled"); present || d.HasChange("enabled") {
		body.Enabled = d.Get("enabled").(bool)
	}
	if _, present := d.GetOk("use_default_ttl"); present || d.HasChange("use_default_ttl") {
		body.UseDefaultTTL = d.Get("use_default_ttl").(bool)
	}
	if _, present := d.GetOk("ttl_queries_ms"); present || d.HasChange("ttl_queries_ms") {
		body.TTLQueriesMs = int64(d.Get("ttl_queries_ms").(int))
	}
	if _, present := d.GetOk("ttl_resources_ms"); present || d.HasChange("ttl_resources_ms") {
		body.TTLResourcesMs = int64(d.Get("ttl_resources_ms").(int))
	}

	_, err := client.Enterprise.SetDataSourceCacheConfig(dsUID, body)
	if err != nil {
		return diag.FromErr(err)
	}

	// Fetch DS to get OrgID for resource ID
	if err := setIDFromDatasource(ctx, d, client, dsUID); err != nil {
		return diag.FromErr(err)
	}
	return ReadDataSourceCacheConfig(ctx, d, meta)
}

func ReadDataSourceCacheConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	// idStr is datasource UID
	resp, err := client.Enterprise.GetDataSourceCacheConfig(idStr)
	if err, shouldReturn := common.CheckReadError("datasource cache config", d, err); shouldReturn {
		return err
	}
	cc := resp.GetPayload()

	// Also read DS to set org_id and ensure ID is correct
	dsResp, err := client.Datasources.GetDataSourceByUID(idStr)
	if err, shouldReturn := common.CheckReadError("datasource", d, err); shouldReturn {
		return err
	}
	ds := dsResp.GetPayload()

	d.Set("datasource_uid", cc.DataSourceUID)
	d.Set("org_id", strconv.FormatInt(ds.OrgID, 10))
	d.Set("enabled", cc.Enabled)
	d.Set("use_default_ttl", cc.UseDefaultTTL)
	d.Set("ttl_queries_ms", int(cc.TTLQueriesMs))
	d.Set("ttl_resources_ms", int(cc.TTLResourcesMs))

	// Ensure ID matches current org/datasource
	d.SetId(MakeOrgResourceID(ds.OrgID, idStr))
	return nil
}

func DeleteDataSourceCacheConfig(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, err := client.Enterprise.DisableDataSourceCache(idStr)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func setIDFromDatasource(ctx context.Context, d *schema.ResourceData, client *goapi.GrafanaHTTPAPI, dataSourceUID string) error {
	resp, err := client.Datasources.GetDataSourceByUID(dataSourceUID)
	if err != nil {
		return err
	}
	ds := resp.GetPayload()
	d.SetId(MakeOrgResourceID(ds.OrgID, ds.UID))
	return nil
}
