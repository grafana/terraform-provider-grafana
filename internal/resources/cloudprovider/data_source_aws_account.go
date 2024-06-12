package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceAWSAccount() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: datasourceAWSAccountRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceAWSAccount().Schema, map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the AWS Account resource to look up.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"role_arn": {
				Description: "The IAM Role ARN string of the AWS Account resource to look up.",
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

func datasourceAWSAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	d.SetId(resourceAWSAccountTerraformID.Make(d.Get("stack_id").(string), d.Get("role_arn").(string)))
	d.Set("role_arn", TestAWSAccountData.RoleARN)
	d.Set("regions", TestAWSAccountData.Regions)

	return diags
}
