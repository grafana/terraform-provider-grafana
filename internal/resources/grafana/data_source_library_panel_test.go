package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanel_basic(t *testing.T) {
	testCases := []struct {
		versionConstraint string
		replacements      map[string]string
	}{
		{
			versionConstraint: ">=8.0.0,<=11.0.0",
			replacements: map[string]string{
				`"general"`: "null",
			},
		},
		{
			versionConstraint: ">11.0.0",
			replacements:      map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			var panel models.LibraryElementResponse
			checks := []resource.TestCheckFunc{
				libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
				resource.TestCheckResourceAttr(
					"data.grafana_library_panel.from_name", "name", "test name",
				),
				resource.TestMatchResourceAttr(
					"data.grafana_library_panel.from_name", "uid", common.UIDRegexp,
				),
				resource.TestCheckResourceAttr(
					"data.grafana_library_panel.from_uid", "name", "test name",
				),
				resource.TestMatchResourceAttr(
					"data.grafana_library_panel.from_uid", "uid", common.UIDRegexp,
				),
			}

			// TODO: Make parallelizable
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             libraryPanelCheckExists.destroyed(&panel, nil),
				Steps: []resource.TestStep{
					{
						Config: testutils.TestAccExampleWithReplace(t,
							"data-sources/grafana_library_panel/data-source.tf",
							tc.replacements,
						),
						Check: resource.ComposeTestCheckFunc(checks...),
					},
				},
			})
		})
	}
}
