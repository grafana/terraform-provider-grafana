package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantTermsAcceptance_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_terms_acceptance/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_terms_acceptance.test", "id", "terms"),
					resource.TestCheckResourceAttr("grafana_assistant_terms_acceptance.test", "accepted", "true"),
				),
			},
			{
				ResourceName:      "grafana_assistant_terms_acceptance.test",
				ImportState:       true,
				ImportStateId:     "terms",
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_terms_acceptance/_acc_basic.tf", map[string]string{
					"accepted = true": "accepted = false",
				}),
				Check: resource.TestCheckResourceAttr("grafana_assistant_terms_acceptance.test", "accepted", "false"),
			},
		},
	})
}
