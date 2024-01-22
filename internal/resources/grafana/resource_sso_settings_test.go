package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestSSOSettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var settings models.SSOSettings

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      ssoSettingsCheckExists.destroyed(&settings, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_sso_settings/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					ssoSettingsCheckExists.exists("grafana_sso_settings.github_sso_settings", &settings),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "provider_name", "github"),
				),
			},
			{
				ResourceName:            "grafana_sso_settings.github_sso_settings",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			{
				ResourceName: "grafana_sso_settings.github_sso_settings",
				ImportState:  true,
			},
		},
	})
}
