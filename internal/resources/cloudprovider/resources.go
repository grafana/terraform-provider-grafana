package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *cloudproviderapi.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).CloudProviderAPI
		if client == nil {
			return diag.Errorf("the Cloud Provider API client is required for this resource. Set the cloud_provider_access_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

var DataSources = []*common.DataSource{
	datasourceAWSAccount(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
	resourceAWSCloudWatchScrapeJob(),
}
