package k6

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var Resources = addValidationToResources(
	resourceProject(),
	resourceProjectLimits(),
	resourceLoadTest(),
)

func addValidationToResources(resources ...*common.Resource) []*common.Resource {
	for _, r := range resources {
		addValidationToSchema(r.Schema)
	}
	return resources
}

func addValidationToSchema(r *schema.Resource) {
	if r == nil {
		return
	}
	createFn := r.CreateContext
	readFn := r.ReadContext
	updateFn := r.UpdateContext
	deleteFn := r.DeleteContext

	if createFn != nil {
		r.CreateContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			if err := k6ClientResourceValidation(d, m); err != nil {
				return diag.FromErr(err)
			}
			return createFn(ctx, d, m)
		}
	}

	if readFn != nil {
		r.ReadContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			if err := k6ClientResourceValidation(d, m); err != nil {
				return diag.FromErr(err)
			}
			return readFn(ctx, d, m)
		}
	}

	if updateFn != nil {
		r.UpdateContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			if err := k6ClientResourceValidation(d, m); err != nil {
				return diag.FromErr(err)
			}
			return updateFn(ctx, d, m)
		}
	}

	if deleteFn != nil {
		r.DeleteContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			if err := k6ClientResourceValidation(d, m); err != nil {
				return diag.FromErr(err)
			}
			return deleteFn(ctx, d, m)
		}
	}
}

func k6ClientResourceValidation(_ *schema.ResourceData, m interface{}) error {
	if m.(*common.Client).K6APIClient == nil || m.(*common.Client).K6APIConfig == nil {
		return fmt.Errorf("the k6 Cloud API client is required for this resource. Set the k6_access_token and k6_stack_id provider attributes")
	}
	return nil
}
