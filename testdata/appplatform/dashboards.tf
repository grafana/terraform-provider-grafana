resource "grafana_apps_dashboard_dashboard_v1alpha1" "test_dashboard_one" {
  metadata {
    uid        = "test_dashboard_one"
    folder_uid = grafana_folder.test_folder_one.uid
  }

  spec {
    json  = file("${path.module}/dashboards/test_dashboard_one.json")
    title = "Test Dashboard One"
    tags = [
      "one",
      "two",
      "three",
    ]
  }

  options {
    overwrite = true
  }
}

resource "grafana_apps_dashboard_dashboard_v1alpha1" "test_dashboard_two" {
  metadata {
    uid        = "test_dashboard_two"
    folder_uid = grafana_folder.test_folder_two.uid
  }

  spec {
    json = file("${path.module}/dashboards/test_dashboard_two.json")
  }

  options {
    overwrite = true
  }
}

resource "grafana_apps_dashboard_dashboard_v2beta1" "test_dashboard_v2" {
  metadata {
    uid        = "test_dashboard_v2"
    folder_uid = grafana_folder.test_folder_one.uid
  }

  spec {
    json  = file("${path.module}/dashboards/test_dashboard_v2.json")
    title = "Test Dashboard V2"
    tags = [
      "v2",
      "test",
    ]
  }

  options {
    overwrite = true
  }
}
