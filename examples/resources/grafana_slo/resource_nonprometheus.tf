resource "grafana_slo" "test" {
  name        = "Terraform Testing"
  description = "Terraform Description"
  query {
    freeform {
      query = jsonencode([
        {
          aggregation : "Sum",
          alias : "",
          application : "57831",
          applicationName : "petclinic",
          datasource : {
            "type" : "dlopes7-appdynamics-datasource",
            "uid" : "appdynamics_localdev"
          },
          delimiter : "|",
          isRawQuery : false,
          metric : "Service Endpoints|PetClinicEastTier1|/petclinic/api_SERVLET|Errors per Minute",
          queryType : "metrics",
          refId : "errors",
          rollUp : true,
          schemaVersion : "3.9.5",
          transformLegend : "Segments",
          transformLegendText : ""
        },
        {
          aggregation : "Sum",
          alias : "",
          application : "57831",
          applicationName : "petclinic",
          datasource : {
            "type" : "dlopes7-appdynamics-datasource",
            "uid" : "appdynamics_localdev"
          },
          intervalMs : 1000,
          maxDataPoints : 43200,
          delimiter : "|",
          isRawQuery : false,
          metric : "Service Endpoints|PetClinicEastTier1|/petclinic/api_SERVLET|Calls per Minute",
          queryType : "metrics",
          refId : "total",
          rollUp : true,
          schemaVersion : "3.9.5",
          transformLegend : "Segments",
          transformLegendText : ""
        },
        {
          datasource : {
            "type" : "__expr__",
            "uid" : "__expr__"
          },
          expression : "($total - $errors) / $total",
          intervalMs : 1000,
          maxDataPoints : 43200,
          refId : "C",
          type : "math"
        }
      ])
    }
    type = "freeform"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
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