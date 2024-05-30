package cloudobservability

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceAWSAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceAWSAccountRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceAWSAccount().Schema, map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "A name to help identify this AWS Account resource. This name must be unique per StackID. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
		}),
	}
}

func datasourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId(resourceAWSAccountTerraformID.Make(d.Get("stack_id").(string), d.Get("name").(string)))
	d.Set("role_arns", TestAWSAccountData.RoleARNs)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}
