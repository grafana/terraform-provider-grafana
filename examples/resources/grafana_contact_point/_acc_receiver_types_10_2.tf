resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v10.2"

  oncall {
    url                 = "http://my-url"
    http_method         = "POST"
    basic_auth_user     = "user"
    basic_auth_password = "password"
    max_alerts          = 100
    message             = "Custom message"
    title               = "Custom title"
  }
  oncall {
    url                       = "http://my-url"
    http_method               = "POST"
    authorization_scheme      = "Basic"
    authorization_credentials = "dXNlcjpwYXNzd29yZA=="
    max_alerts                = 100
    message                   = "Custom message"
    title                     = "Custom title"
  }
}
