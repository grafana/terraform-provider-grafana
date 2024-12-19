resource "grafana_message_template" "my_template" {
  name     = "My Notification Template Group"
  template = "{{define \"custom.message\" }}\n template content\n{{ end }}"
}
