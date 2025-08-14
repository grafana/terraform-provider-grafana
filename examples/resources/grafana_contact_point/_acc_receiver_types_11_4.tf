resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v11.4"

  webhook {
    url                 = "http://my-url"
    http_method         = "POST"
    basic_auth_user     = "user"
    basic_auth_password = "password"
    max_alerts          = 100
    message             = "Custom message"
    title               = "Custom title"
    tls_config = {
      insecure_skip_verify = true
      ca_certificate       = "ca.crt"
      client_certificate   = "client.crt"
      client_key           = "client.key"
    }
  }

  webhook {
    url                       = "http://my-url"
    http_method               = "POST"
    authorization_scheme      = "Basic"
    authorization_credentials = "dXNlcjpwYXNzd29yZA=="
    max_alerts                = 100
    message                   = "Custom message"
    title                     = "Custom title"
    tls_config = {
      insecure_skip_verify = true
      ca_certificate       = "ca.crt"
      client_certificate   = "client.crt"
      client_key           = "client.key"
    }
  }
}
