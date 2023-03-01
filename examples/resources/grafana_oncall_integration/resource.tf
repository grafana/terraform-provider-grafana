resource "grafana_oncall_integration" "test-acc-integration" {
  provider = grafana.oncall
  name     = "my integration"
  type     = "grafana"
  default_route {
  }
}

# Also it's possible to manage integration templates.
# Check docs to see all available templates.
resource "grafana_oncall_integration" "integration_with_templates" {
  provider = grafana.oncall
  name     = "integration_with_templates"
  type     = "webhook"
  default_route {
  }
  templates {
    grouping_key = "{{ payload.group_id }}"
    slack {
      title     = "Slack title"
      message   = <<-EOT
          This is example of multiline template
          {{ payload.message }}
        EOT
      image_url = "{{ payload.image_url }}"
    }
  }
}
