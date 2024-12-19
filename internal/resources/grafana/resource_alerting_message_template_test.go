package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMessageTemplate_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var tmpl models.NotificationTemplate

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingMessageTemplateCheckExists.destroyed(&tmpl, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_message_template/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Notification Template Group"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"custom.message\" }}\n template content\n{{ end }}"),
					testutils.CheckLister("grafana_message_template.my_template"),
				),
			},
			// Test import.
			{
				ResourceName:            "grafana_message_template.my_template",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"disable_provenance"},
			},
			// Test update with heredoc template doesn't change
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					`template = "{{define \"custom.message\" }}\n template content\n{{ end }}"`: `template = <<-EOT
{{define "custom.message" }}
 template content
{{ end }}
EOT`,
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Notification Template Group"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"custom.message\" }}\n template content\n{{ end }}"),
				),
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					"template content": "different content",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "My Notification Template Group"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"custom.message\" }}\n different content\n{{ end }}"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_message_template/resource.tf", map[string]string{
					"My Notification Template Group": "A Different Template",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "A Different Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"custom.message\" }}\n template content\n{{ end }}"),
					alertingMessageTemplateCheckExists.destroyed(&models.NotificationTemplate{Name: "My Notification Template Group"}, nil),
				),
			},
		},
	})
}

func TestAccMessageTemplate_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	name := acctest.RandString(10)
	var tmpl models.NotificationTemplate
	var org models.OrgDetailsDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccMessageTemplate_inOrg(name),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					checkResourceIsInOrg("grafana_message_template.my_template", "grafana_organization.test"),
					resource.TestMatchResourceAttr("grafana_message_template.my_template", "id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "my-template"),
				),
			},
			// Test import.
			{
				ResourceName:            "grafana_message_template.my_template",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"disable_provenance"},
			},
			// Test delete template in org.
			{
				Config: testutils.WithoutResource(t, testAccMessageTemplate_inOrg(name), "grafana_message_template.my_template"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingMessageTemplateCheckExists.destroyed(&tmpl, &org),
				),
			},
		},
	})
}

func testAccMessageTemplate_inOrg(name string) string {
	return fmt.Sprintf(`
	resource "grafana_organization" "test" {
		name = "%[1]s"
	}

	resource "grafana_message_template" "my_template" {
		org_id = grafana_organization.test.id
		name = "my-template"
		template = "{{define \"custom.message\" }}\n template content\n{{ end }}"
	}
	`, name)
}
