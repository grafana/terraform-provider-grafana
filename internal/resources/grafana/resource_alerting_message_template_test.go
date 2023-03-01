package grafana_test

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccMessageTemplate_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0")

	var tmpl gapi.AlertingMessageTemplate

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testMessageTemplateCheckDestroy(&tmpl),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_message_template/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testMessageTemplateCheckExists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Reusable Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"My Reusable Template\" }}\n template content\n{{ end }}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_message_template.my_template",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update with heredoc template doesn't change
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					`template = "{{define \"My Reusable Template\" }}\n template content\n{{ end }}"`: `template = <<-EOT
{{define "My Reusable Template" }}
 template content
{{ end }}
EOT`,
				}),
				Check: resource.ComposeTestCheckFunc(
					testMessageTemplateCheckExists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Reusable Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"My Reusable Template\" }}\n template content\n{{ end }}"),
				),
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					"template content": "different content",
				}),
				Check: resource.ComposeTestCheckFunc(
					testMessageTemplateCheckExists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Reusable Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"My Reusable Template\" }}\n different content\n{{ end }}"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					"My Reusable Template": "A Different Template",
				}),
				Check: resource.ComposeTestCheckFunc(
					testMessageTemplateCheckExists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "A Different Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"A Different Template\" }}\n template content\n{{ end }}"),
					testMessageTemplateCheckDestroy(&gapi.AlertingMessageTemplate{Name: "My Reusable Template"}),
				),
			},
		},
	})
}

func testMessageTemplateCheckExists(rname string, mt *gapi.AlertingMessageTemplate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		tmpl, err := client.MessageTemplate(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting resource: %s", err)
		}

		*mt = *tmpl
		return nil
	}
}

func testMessageTemplateCheckDestroy(mt *gapi.AlertingMessageTemplate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		tmpl, err := client.MessageTemplate(mt.Name)
		if err == nil && tmpl != nil {
			return fmt.Errorf("message template still exists on the server")
		}
		return nil
	}
}
