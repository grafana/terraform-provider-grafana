package slo

import (
	"context"

	slo "github.com/grafana/slo-openapi-client/go"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *slo.APIClient) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).SLOClient
		if client == nil {
			return diag.Errorf("the SLO API client is required for this resource. Set the url and auth provider attributes")
		}
		return f(ctx, d, client)
	}
}

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_slos": datasourceSlo(),
}

var Resources = []*common.Resource{
	resourceSlo(),
}
