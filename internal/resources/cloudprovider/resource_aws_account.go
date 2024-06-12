package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourceAWSAccountTerraformID = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("role_arn"))
)

func resourceAWSAccount() *common.Resource {
	schema := &schema.Resource{
		CreateContext: withClient[schema.CreateContextFunc](resourceAWSAccountCreate),
		ReadContext:   resourceAWSAccountRead,
		UpdateContext: resourceAWSAccountUpdate,
		DeleteContext: resourceAWSAccountDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID computed by the Grafa Cloud Provider API for the AWS Account resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"stack_id": {
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"role_arn": {
				Description: "An IAM Role ARN string to represent with this AWS Account resource.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A list of regions that this AWS Account resource applies to.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_account",
		resourceAWSAccountTerraformID,
		schema,
	)
}

func resourceAWSAccountCreate(ctx context.Context, d *schema.ResourceData, c *cloudproviderapi.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	account, err := c.CreateAWSAccount(
		ctx,
		d.Get("stack_id").(string),
		cloudproviderapi.AWSAccount{
			RoleARN: d.Get("role_arn").(string),
			Regions: common.ListToStringSlice(d.Get("regions").(*schema.Set).List()),
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(account.ID)
	return diags
}

func resourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.Set("stack_id", TestAWSAccountData.StackID)
	d.Set("role_arn", TestAWSAccountData.RoleARN)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}

func resourceAWSAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	TestAWSAccountData.StackID = d.Get("stack_id").(string)
	TestAWSAccountData.RoleARN = d.Get("role_arn").(string)
	TestAWSAccountData.Regions = d.Get("regions").([]string)
	d.Set("stack_id", TestAWSAccountData.StackID)
	d.Set("role_arn", TestAWSAccountData.RoleARN)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}

func resourceAWSAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId("")

	return diags
}
