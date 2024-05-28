package cloudobservability

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourceAWSAccountID = common.NewResourceID(common.IntIDField("id"))
)

func resourceAWSAccount() *common.Resource {
	schema := &schema.Resource{
		CreateContext: resourceAWSAccountCreate,
		ReadContext:   resourceAWSAccountRead,
		UpdateContext: resourceAWSAccountUpdate,
		DeleteContext: resourceAWSAccountDelete,
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"external_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}

	return common.NewLegacySDKResource(
		"grafana_cloud_observability_aws_account",
		resourceAWSAccountID,
		schema,
	)
}

func resourceAWSAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	// Get the account ID, external ID, and role name from the resource data
	accountID := d.Get("account_id").(string)
	externalID := d.Get("external_id").(string)
	roleName := d.Get("role_name").(string)

	// Create a unique ID for the resource
	d.SetId(fmt.Sprintf("%s:%s:%s", accountID, externalID, roleName))

	return diags
}

func resourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	// Get the account ID, external ID, and role name from the resource data
	accountID := d.Get("account_id").(string)
	externalID := d.Get("external_id").(string)
	roleName := d.Get("role_name").(string)

	// Create a unique ID for the resource
	d.SetId(fmt.Sprintf("%s:%s:%s", accountID, externalID, roleName))

	return diags
}

func resourceAWSAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	// Get the account ID, external ID, and role name from the resource data
	accountID := d.Get("account_id").(string)
	externalID := d.Get("external_id").(string)
	roleName := d.Get("role_name").(string)

	// Create a unique ID for the resource
	d.SetId(fmt.Sprintf("%s:%s:%s", accountID, externalID, roleName))

	return diags
}

func resourceAWSAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	// Clear the ID
	d.SetId("")

	return diags
}
