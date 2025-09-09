resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v12.0"

  webhook {
    url = "http://my-url"
    headers = {
      Content-Type  = "test-content-type"
      X-Test-Header = "test-header-value"
    }
    payload {
      template = "{{ .Receiver }}: {{ .Vars.var1 }}"
      vars = {
        var1 = "variable value"
      }
    }
  }
}
