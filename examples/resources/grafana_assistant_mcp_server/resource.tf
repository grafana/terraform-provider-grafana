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

# Authenticate a GitHub MCP server with a GitHub App installation instead of a personal access token.
resource "grafana_assistant_mcp_server" "github" {
  name         = "GitHub"
  scope        = "tenant"
  applications = ["assistant"]

  configuration {
    url = "https://api.githubcopilot.com/mcp/"
  }

  github_app {
    app_id          = 123456
    installation_id = 12345678
    private_key     = var.github_app_private_key
  }
}

variable "github_app_private_key" {
  type      = string
  sensitive = true
}
