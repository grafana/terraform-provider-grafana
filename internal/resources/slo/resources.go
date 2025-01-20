package slo

import (
	"context"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*client.Client).SLOClient
		if client == nil {
			return diag.Errorf("the SLO API client is required for this resource. Set the url and auth provider attributes")
		}
		return f(ctx, d, client)
	}
}

var DataSources = []*common.DataSource{
	datasourceSlo(),
}

var Resources = []*common.Resource{
	resourceSlo(),
}
