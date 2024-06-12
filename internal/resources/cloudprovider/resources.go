package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestAWSAccountData is only temporarily exported here until
// we have the resource handlers talking to the real API.
// TODO(tristan): move this to test package and unexport
// once we're using the actual API for interactions.
var TestAWSAccountData = struct {
	StackID string
	RoleARN string
	Regions []string
}{
	StackID: "001",
	RoleARN: "arn:aws:iam::123456789012:role/my-role-1a",
	Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
}

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *cloudproviderapi.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).CloudProviderAPI
		if client == nil {
			return diag.Errorf("the Cloud Provider API client is required for this resource. Set the sm_access_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

var Datasources = []*common.DataSource{
	datasourceAWSAccount(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
}
