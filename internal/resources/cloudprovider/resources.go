package cloudprovider

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
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

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_cloud_provider_aws_account": datasourceAWSAccount(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
}
