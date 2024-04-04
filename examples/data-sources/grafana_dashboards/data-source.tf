resource "grafana_organization" "test" {
  name = "testing dashboards data source"
}

resource "grafana_folder" "data_source_dashboards" {
  org_id = grafana_organization.test.id

  title = "test folder data_source_dashboards"
}

resource "grafana_dashboard" "data_source_dashboards1" {
  org_id = grafana_organization.test.id

  folder = grafana_folder.data_source_dashboards.id
  config_json = jsonencode({
    uid   = "data-source-dashboards-1"
    title = "data_source_dashboards 1"
    tags  = ["dev"]
  })
}

resource "grafana_dashboard" "data_source_dashboards2" {
  org_id = grafana_organization.test.id

  config_json = jsonencode({
    uid   = "data-source-dashboards-2"
    title = "data_source_dashboards 2"
    tags  = ["prod"]
  })
}

data "grafana_dashboards" "tags" {
  org_id = grafana_organization.test.id
  tags   = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

data "grafana_dashboards" "folder_uids" {
  org_id      = grafana_organization.test.id
  folder_uids = [grafana_dashboard.data_source_dashboards1.folder]
}

data "grafana_dashboards" "folder_uids_tags" {
  org_id      = grafana_organization.test.id
  folder_uids = [grafana_dashboard.data_source_dashboards1.folder]
  tags        = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

// use depends_on to wait for dashboard resource to be created before searching
data "grafana_dashboards" "all" {
  org_id = grafana_organization.test.id
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}

// get only one result
data "grafana_dashboards" "limit_one" {
  org_id = grafana_organization.test.id
  limit  = 1
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}

// The dashboards are not in the default org so this should return an empty list
data "grafana_dashboards" "wrong_org" {
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}
