package cloudobservability

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO(tristan): move this to test package once we're using
// the actual API for interactions.
var TestAWSAccountData = struct {
	StackID  string
	Name     string
	RoleARNs map[string]string
	Regions  []string
}{
	StackID: "001",
	Name:    "my-aws-account",
	RoleARNs: map[string]string{
		"my role 1a": "arn:aws:iam::123456789012:role/my-role-1a",
		"my role 1b": "arn:aws:iam::123456789012:role/my-role-1b",
		"my role 2":  "arn:aws:iam::210987654321:role/my-role-2",
	},
	Regions: []string{"us-east-1", "us-east-2", "us-west-1"},
}

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_cloud_observability_aws_account": datasourceAWSAccount(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
}
