terraform {
  required_providers {
    grafana = {
      version = "999.999.999"
      source  = "grafana/grafana"
    }
  }
}

provider "grafana" {
  url  = "http://localhost:3000"
  auth = "admin:admin"
}

resource "grafana_folder" "my_folder" {
  title = "My Folder"
}
