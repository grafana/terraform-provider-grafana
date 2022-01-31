package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceStack(t *testing.T) {
	CheckCloudTestsEnabled(t)

	var stack gapi.Stack
	checks := []resource.TestCheckFunc{
		testAccStackCheckExists("grafana_cloud_stack.test", &stack),
		resource.TestCheckResourceAttr(
			"data.grafana_cloud_stack.test", "sllug", "test-slug",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_cloud_stack.test", "name", "test-stack",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_cloud_stack.test", "id", idRegexp,
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_cloud_stack/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
