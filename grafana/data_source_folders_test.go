package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolders(t *testing.T) {
	CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr("data.grafana_folders.one", "folders.%", "1"),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_folders/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
