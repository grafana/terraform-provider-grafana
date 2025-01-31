resource "grafana_slo" "big_tent_test" {
  name        = "Terraform Testing - Big Tent"
  description = "Terraform Description - Big Tent"
  query {
    freeform {
      query = jsonencode([
{
aggregation: "Sum",
alias: "",
application: "57831",
applicationName: "petclinic",
datasource: {
type: "dlopes7-appdynamics-datasource",
uid: "appdynamics_localdev"
},
delimiter: "|",
isRawQuery: false,
metric: "Overall Application Performance|Calls per Minute",
queryType: "metrics",
refId: "total",
rollUp: true,
schemaVersion: "3.9.5",
transformLegend: "Segments",
transformLegendText: ""
},
{
aggregation: "Sum",
alias: "",
application: "57831",
applicationName: "petclinic",
datasource: {
type: "dlopes7-appdynamics-datasource",
uid: "appdynamics_localdev"
},
intervalMs: 1000,
maxDataPoints:43200,
delimiter: "|",
isRawQuery: false,
metric: "Overall Application Performance|Calls per Minute",
queryType: "metrics",
refId: "also_total",
rollUp: true,
schemaVersion: "3.9.5",
transformLegend: "Segments",
transformLegendText: ""
},
{
conditions: [
{
evaluator: {
params: [
0,
0
],
type: "gt"
},
operator: {
type: "and"
},
query: {
params: []
},
reducer: {
params: [],
type: "avg"
},
type: "query"
}
],
datasource: {
name: "Expression",
type: "__expr__",
uid: "__expr__"
},
expression: "($total / $also_total)",
intervalMs: 1000,
maxDataPoints: 43200,
refId: "C",
type: "math"
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