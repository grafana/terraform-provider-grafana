package cloudobservability

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourceAWSAccountTerraformID = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("name"))
)

func resourceAWSAccount() *common.Resource {
	schema := &schema.Resource{
		CreateContext: resourceAWSAccountCreate,
		ReadContext:   resourceAWSAccountRead,
		UpdateContext: resourceAWSAccountUpdate,
		DeleteContext: resourceAWSAccountDelete,
		Schema: map[string]*schema.Schema{
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
			"role_arns": {
				Description: "A map consisting of one or more name => IAM Role ARN pairs to associate with this AWS Account resource.",
				Type:        schema.TypeMap,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"regions": {
				Description: "A list of regions that this AWS Account resource applies to.",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		"grafana_cloud_observability_aws_account",
		resourceAWSAccountTerraformID,
		schema,
	)
}

func resourceAWSAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId(resourceAWSAccountTerraformID.Make(TestAWSAccountData.StackID, TestAWSAccountData.Name))

	return resourceAWSAccountRead(ctx, d, nil)
}

func resourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.Set("role_arns", TestAWSAccountData.RoleARNs)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}

func resourceAWSAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	TestAWSAccountData.RoleARNs = d.Get("role_arns").(map[string]string)
	TestAWSAccountData.Regions = d.Get("regions").([]string)
	d.Set("role_arns", TestAWSAccountData.RoleARNs)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}

func resourceAWSAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId("")

	return diags
}
