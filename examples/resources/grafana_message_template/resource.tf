resource "grafana_message_template" "my_template" {
    name = "My Reusable Template"
    template = "{{define \"My Reusable Template\" }}\n template content\n{{ end }}"
}
