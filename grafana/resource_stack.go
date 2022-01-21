package grafana

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceStack() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#stacks/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/stack/)
`,

		CreateContext: CreateStack,
		UpdateContext: UpdateStack,
		DeleteContext: DeleteStack,
		Exists:        ExistsStack,
		ReadContext:   ReadStack,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"stack_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. “<stack_slug>.grafana.net”).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of stack.",
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: `
Subdomain that the Grafana instance will be available at (i.e. setting slug to “<stack_slug>” will make the instance
available at “https://<stack_slug>.grafana.net".`,
			},
			"region_slug": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Region slug to assign to this stack.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the Grafana instance.",
			},
			"org_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the stack.",
			},
			"cluster_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Cluster id to assign to this stack.",
			},
			"cluster_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster name assigned to this stack.",
			},
			"cluster_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster slug assigned to this stack.",
			},
			"prom_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Id of the instance that is used as a health monitor.",
			},
			"prom_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prom_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prom_status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},
			"graphite_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Graphite url for this instance.",
			},
			"graphite_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Graphite name for this instance.",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Region id to assign to this stack.",
			},
		},
	}
}

func CreateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	name := d.Get("name").(string)
	slug := d.Get("slug").(string)
	region := d.Get("region_slug").(string)

	stackID, err := client.NewStack(name, slug, region)
	if err != nil && err.Error() == "409 Conflict" {
		return diag.Errorf("Error: A Grafana stack with the name '%s' already exists.", name)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(stackID, 10))

	return diag.Diagnostics{}
}

func ReadStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	stack, err := client.StackByID(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing stack %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(stack.ID, 10))
	d.Set("name", stack.Name)
	d.Set("slug", stack.Slug)
	d.Set("org_id", stack.OrgID)
	d.Set("org_slug", stack.OrgSlug)
	d.Set("org_name", stack.OrgName)
	d.Set("url", stack.URL)
	d.Set("status", stack.Status)
	d.Set("cluster_id", stack.ClusterID)
	d.Set("cluster_name", stack.ClusterName)
	d.Set("cluster_slug", stack.ClusterSlug)
	d.Set("prom_id", stack.HmInstancePromID)
	d.Set("prom_url", stack.HmInstancePromURL)
	d.Set("prom_name", stack.HmInstancePromName)
	d.Set("prom_status", stack.HmInstancePromStatus)
	d.Set("graphite_url", stack.HmInstanceGraphiteURL)
	d.Set("graphite_name", stack.HmInstanceGraphiteName)
	d.Set("region_id", stack.RegionID)
	d.Set("region_slug", stack.RegionSlug)

	return nil
}

func UpdateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	stackID, _ := strconv.ParseInt(d.Id(), 10, 64)

	// The underlying API olnly allows to update the name and description.
	allowed_changes := []string{"name", "description"}
	if d.HasChangesExcept(allowed_changes...) {
		return diag.Errorf("Error: Only name and description can be updated.")
	}

	if d.HasChange("name") || d.HasChange("description") {
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		err := client.UpdateStack(stackID, name, description)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
}

func DeleteStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	slug := d.Get("slug").(string)
	if err := client.DeleteStack(slug); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ExistsStack(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*client).gapi
	stackID, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.StackByID(stackID)
	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}
