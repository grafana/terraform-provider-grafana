data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "dns" {
  job     = "DNS Updated"
  target  = "grafana.net"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Frankfurt,
    data.grafana_synthetic_monitoring_probes.main.probes.London,
  ]
  labels = {
    foo = "baz"
  }
  settings {
    dns {
      ip_version  = "Any"
      server      = "8.8.4.4"
      port        = 8600
      record_type = "CNAME"
      protocol    = "TCP"
      valid_r_codes = [
        "NOERROR",
        "NOTAUTH",
      ]
      validate_answer_rrs {
        fail_if_matches_regexp = [
          ".+-bad-stuff*",
        ]
        fail_if_not_matches_regexp = [
          ".+-good-stuff*",
        ]
      }
      validate_authority_rrs {
        fail_if_matches_regexp = [
          ".+-bad-stuff*",
        ]
        fail_if_not_matches_regexp = [
          ".+-good-stuff*",
        ]
      }
      validate_additional_rrs {
        fail_if_matches_regexp = [
          ".+-bad-stuff*",
        ]
        fail_if_not_matches_regexp = [
          ".+-good-stuff*",
        ]
      }
    }
  }
}
