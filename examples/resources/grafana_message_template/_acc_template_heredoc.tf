resource "grafana_message_template" "heredoc_template" {
  name     = "Heredoc Notification Template Group"
  template = <<-EOT
{{define "custom.message" }}
 template content
{{ end }}
EOT
}
