package slo_test

import (
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutils.TestAccExample(t, "data-sources/grafana_slo/data-source.tf"),
				ExpectError: regexp.MustCompile(`No SLOs Exist`),
			},
		},
	})
}
