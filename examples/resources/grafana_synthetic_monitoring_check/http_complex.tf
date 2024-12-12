data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "http" {
  job     = "HTTP Defaults"
  target  = "https://grafana.org"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Bangalore,
    data.grafana_synthetic_monitoring_probes.main.probes.Mumbai,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    http {
      ip_version                     = "V6"
      method                         = "TRACE"
      body                           = "and spirit"
      no_follow_redirects            = true
      bearer_token                   = "asdfjkl;"
      proxy_url                      = "https://almost-there"
      fail_if_ssl                    = true
      fail_if_not_ssl                = true
      compression                    = "deflate"
      cache_busting_query_param_name = "pineapple"

      tls_config {
        server_name = "grafana.org"
        client_cert = <<EOS
-----BEGIN CERTIFICATE-----
MIIEljCCAn4CCQCKJPUQQxeO0zANBgkqhkiG9w0BAQsFADANMQswCQYDVQQGEwJT
RTAeFw0yMTA1MjkxOTIyNTdaFw0yNDAzMTgxOTIyNTdaMA0xCzAJBgNVBAYTAlNF
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAnmbazDNUT0rSI4BpGZK+
0AJ+9FDkIYWJUtRLJoxw8CF+AobMFploYA2L2Myt80cTA1w8FrewjC8qlqdnrPWr
h1ely2zsUljgi1/niH0ndjFzliL7UkinXQiAsTtYOrOQmzyd/o5PNdu7dz0m7stD
BN/Sz5TlXZnA1/eJbqV/kqMau6b1MaBx8SbRfUG9+cSmUobFJwuktDrPuwJhcEkl
iDmhEqu1GuZzmKvzPacLTVia1vSlmCTCu89NiHI8iGiiLtqNrapup7f8j5m3a3SL
a+vXhplFj2piNl7Nc0dfuVgtEliTI+qUL2/+4A7gzRWZpHy21/LxMMXmBhdJW9En
FWkev97VZLgb5TR3+qpSWmXcodjPy4dibvwsOMpdd+Q4AYulwvlDw5idRPVgGvk7
qq03+w9ppZ5Fugws9k2CD9F/75JX2mCbRpkuPe8XXZ7bqrMaQgQMLOrs68HuiiCk
FTklglq4DMKxnf/Y/T/MgIa9Q1o28YSevh6A7FnfPGARj2H2T4rToi+bC1Vf7qNB
Z18bDpz99tRUTbyiRUSBMWLCGhU6c4HAqUrfrkpperOKFBQ3i38a79838oFdXHBW
6rx1t5cC3XwtEoUyeBKAygez8G1LDXbN3607MxVhAjhHKtPkYvuBfysSNU6JrR0z
UV1IURJANt2UMuKgSEkG/IMCAwEAATANBgkqhkiG9w0BAQsFAAOCAgEAcipMhp/w
yzfPy61faVAw9SPaMNRlnW9FCDC3N9CGOjo2knjXpObPzyzsJiUURTjrA9eFMpRA
e2Rgn2j+nvm2XdLAlC4Kh8jqv/wCL0X6BTQMdN5aOhXdSiXtpXOMvXYY/dQ4ebRZ
XeRCVWQD79JbV6/uyx0nCV3FVcU7L1P4UjxroefVr0soLPMirgxHmOxLnkoVgdcB
tqufP5kJx9CIeJXPx3QQsk1XfEtxtUvuw4ZaZkQnNUqvGl7V+AZpur5Eqfv3zBi8
QxxL7qGkARNssNWH2Ju+tqpM/UZRnjlFrDR4SXUgT0coTduBalUY6qHkciHmRpiP
tf3SgpDeiCSOV2iVFGdaR1mz3muWoAYWFstcWN3a3HjjVugIi23yLN8Gv8CNeoH4
prulinFCLrFgAh8SLAF8mOAZanT06LH8jOIFYrdUxH+ZeRBR0rLoFjUF+JB7UKD9
5TA+B4EBzQ1tMbGFU1DX79MjAejq0IV0Nzq+GMfBvLHxEf4+Oz8nqhDXQcJ6TdtY
l3Lyw5zBvOL80SBK+Mr0UP7d9U3VXgbGHCYVJU6Ot1TwiGwahtWALRALA3TWeGkq
7kyD1H+nm+9lfKhuyBRQnRGBVyze2lAp7oxwshJuhBwEXosXFxq1Cy6QhPN77r6N
vuhxvtppolNnyOgGxwG4zquqq2V5/+vKjKY=
-----END CERTIFICATE-----
EOS
      }

      headers = [
        "Content-Type: multipart/form-data; boundary=something",
      ]

      basic_auth {
        username = "open"
        password = "sesame"
      }

      valid_status_codes = [
        200,
        201,
      ]

      valid_http_versions = [
        "HTTP/1.0",
        "HTTP/1.1",
        "HTTP/2.0",
      ]

      fail_if_body_matches_regexp = [
        ".*bad stuff.*",
      ]

      fail_if_body_not_matches_regexp = [
        ".*good stuff.*",
      ]

      fail_if_header_matches_regexp {
        header        = "Content-Type"
        regexp        = "application/soap*"
        allow_missing = true
      }
    }
  }
}
