package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMessageTemplate_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var tmpl models.NotificationTemplate

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingMessageTemplateCheckExists.destroyed(&tmpl, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_message_template/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
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
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
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
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
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
					alertingMessageTemplateCheckExists.exists("grafana_message_template.my_template", &tmpl),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "name", "A Different Template"),
					resource.TestCheckResourceAttr("grafana_message_template.my_template", "template", "{{define \"A Different Template\" }}\n template content\n{{ end }}"),
					alertingMessageTemplateCheckExists.destroyed(&models.NotificationTemplate{Name: "My Reusable Template"}, nil),
				),
			},
		},
	})
}
