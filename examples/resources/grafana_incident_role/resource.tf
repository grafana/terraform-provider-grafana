resource "grafana_incident_role" "commander" {
  name        = "Commander"
  description = "Coordinates the response and owns the incident."
  important   = true
  mandatory   = true
}
