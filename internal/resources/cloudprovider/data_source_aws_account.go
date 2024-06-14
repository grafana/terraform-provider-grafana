package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceAWSAccount() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: withClient[schema.ReadContextFunc](datasourceAWSAccountRead),
		Schema: common.CloneResourceSchemaForDatasource(resourceAWSAccount().Schema, map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the AWS Account resource to look up.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"account_id": {
				Description: "The ID computed by the Grafana Cloud Provider API for the AWS Account resource.",
				Type:        schema.TypeString,
				Required:    true,
			},
		}),
	}

	return common.NewLegacySDKDataSource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_account",
		schema,
	)
}

func datasourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, c *cloudproviderapi.Client) diag.Diagnostics {
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
