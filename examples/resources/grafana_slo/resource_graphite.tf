resource "grafana_slo" "test" {
  name        = "Terraform Testing"
  description = "Terraform Description"
  query {
    grafana_queries {
      grafana_queries = jsonencode([
        {
          datasource : {
            "type" : "graphite",
            "uid" : "datasource-uid"
          },
          refId : "Success",
          target : "groupByNode(perSecond(web.*.http.2xx_success.*.*), 3, 'avg')"
        },
        {
          datasource : {
            "type" : "graphite",
            "uid" : "datasource-uid"
          },
          refId : "Total",
          target : "groupByNode(perSecond(web.*.http.5xx_errors.*.*), 3, 'avg')"
        },
        {
          datasource : {
            "type" : "__expr__",
            "uid" : "__expr__"
          },
          expression : "$Success / $Total",
          refId : "Expression",
          type : "math"
        }
      ])
    }
    type = "grafana_queries"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }

  label {
    key   = "slo"
    value = "terraform"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate Very High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }
  }
}