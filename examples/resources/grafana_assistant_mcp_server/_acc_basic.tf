resource "grafana_assistant_mcp_server" "test" {
  name         = "tf-acc-test-mcp"
  scope        = "tenant"
  applications = ["assistant"]

  configuration {
    url = "https://httpbin.org/anything"
  }

  custom_headers = {
    Authorization = "Bearer test-token"
  }
}
