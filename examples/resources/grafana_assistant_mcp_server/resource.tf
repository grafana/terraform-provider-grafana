resource "grafana_assistant_mcp_server" "example" {
  name         = "Example MCP server"
  scope        = "tenant"
  applications = ["assistant"]

  configuration {
    url = "https://example.com/mcp/"
  }

  custom_headers = {
    Authorization = "Bearer ${var.mcp_token}"
  }
}

variable "mcp_token" {
  type      = string
  sensitive = true
}
