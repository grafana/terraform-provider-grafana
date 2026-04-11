data "grafana_data_sources" "all" {}

output "data_source_names" {
  value = [for ds in data.grafana_data_sources.all.data_sources : ds.name]
}
