data "grafana_data_sources" "available" {}

variable "alerts" {
  type = map(object({
    data_source_name = string
  }))
  default = {}
}

locals {
  data_sources_by_name = {
    for source in data.grafana_data_sources.available.data_sources : source.name => source.uid
  }

  eligible_alerts = {
    for name, alert in var.alerts : name => alert
    if contains(keys(local.data_sources_by_name), alert.data_source_name)
  }
}

output "eligible_alerts" {
  value = local.eligible_alerts
}