package machinelearning

import (
	"context"
	"errors"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func lister(f func(ctx context.Context, client *mlapi.Client) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *client.Client, data any) ([]string, error) {
		if client.MLAPI == nil {
			return nil, errors.New("the ML API client is required for this resource. Set the url and auth provider attributes")
		}
		return f(ctx, client.MLAPI)
	}
}

func checkClient(f func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*client.Client).MLAPI
		if client == nil {
			return diag.Errorf("the ML API client is required for this resource. Set the url and auth provider attributes")
		}
		return f(ctx, d, meta)
	}
}

var DataSources = []*common.DataSource{}

var Resources = []*common.Resource{
	resourceJob(),
	resourceHoliday(),
	resourceOutlierDetector(),
	resourceAlert(),
}
