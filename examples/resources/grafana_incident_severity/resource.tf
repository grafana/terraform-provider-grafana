resource "grafana_incident_severity" "informational" {
  label       = "Informational"
  level       = 4
  description = "No customer impact, tracked for awareness."
  icon        = "angle-double-down"
  dark_color  = "rgba(110, 159, 255, 1)"
  light_color = "rgba(31, 98, 224, 1)"
}
