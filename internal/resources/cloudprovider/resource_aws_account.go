package cloudprovider

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	resourceAWSAccountTerraformID = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("account_id"))
)

func resourceAWSAccount() *common.Resource {
	schema := &schema.Resource{
		CreateContext: withClient[schema.CreateContextFunc](resourceAWSAccountCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceAWSAccountRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceAWSAccountUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceAWSAccountDelete),
		Schema: map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"account_id": {
				Description: "The ID computed by the Grafana Cloud Provider API for the AWS Account resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"role_arn": {
				Description: "An IAM Role ARN string to represent with this AWS Account resource.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "A set of regions that this AWS Account resource applies to.",
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
	stackID := d.Get("stack_id").(string)
	account, err := c.CreateAWSAccount(
		ctx,
		stackID,
		cloudproviderapi.AWSAccount{
			RoleARN: d.Get("role_arn").(string),
			Regions: common.SetToStringSlice(d.Get("regions").(*schema.Set)),
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("account_id", account.ID)
	d.SetId(fmt.Sprintf("%s-%s", stackID, account.ID))
	return diags
}

func resourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, c *cloudproviderapi.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	account, err := c.GetAWSAccount(
		ctx,
		d.Get("stack_id").(string),
		d.Get("account_id").(string),
	)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("role_arn", account.RoleARN)
	d.Set("regions", common.StringSliceToSet(account.Regions))
	return diags
}

func resourceAWSAccountUpdate(ctx context.Context, d *schema.ResourceData, c *cloudproviderapi.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	_, err := c.UpdateAWSAccount(
		ctx,
		d.Get("stack_id").(string),
		d.Get("account_id").(string),
		cloudproviderapi.AWSAccount{
			RoleARN: d.Get("role_arn").(string),
			Regions: common.SetToStringSlice(d.Get("regions").(*schema.Set)),
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceAWSAccountDelete(ctx context.Context, d *schema.ResourceData, c *cloudproviderapi.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	err := c.DeleteAWSAccount(
		ctx,
		d.Get("stack_id").(string),
		d.Get("account_id").(string),
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
