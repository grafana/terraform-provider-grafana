resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v12.1"

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
    http_config {
      oauth2 {
        client_id     = "client_id"
        client_secret = "client_secret"
        token_url     = "http://oauth2-token-url"
        scopes        = ["scope1", "scope2"]
        endpoint_params = {
          "param1" = "value1"
          "param2" = "value2"
        }
        proxy_config {
          proxy_url = "http://proxy-url"
          proxy_from_environment = false
          no_proxy = "localhost"
          proxy_connect_header = {
            "X-Proxy-Header" = "proxy-value"
          }
        }
        tls_config {
          insecure_skip_verify = true
          ca_certificate       = <<EOF
-----BEGIN CERTIFICATE-----
MIGrMF+gAwIBAgIBATAFBgMrZXAwADAeFw0yNDExMTYxMDI4MzNaFw0yNTExMTYx
MDI4MzNaMAAwKjAFBgMrZXADIQCf30GvRnHbs9gukA3DLXDK6W5JVgYw6mERU/60
2M8+rjAFBgMrZXADQQCGmeaRp/AcjeqmJrF5Yh4d7aqsMSqVZvfGNDc0ppXyUgS3
WMQ1+3T+/pkhU612HR0vFd3vyFhmB4yqFoNV8RML
-----END CERTIFICATE-----
EOF
          client_certificate   = <<EOF
-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----
EOF
          client_key           = <<EOF
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----
EOF
        }
      }
    }
  }
}
