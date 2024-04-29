provider "grafana" {
  auth = "admin:admin"
  http_headers = {
    header1 = "val2"
  }
  url = "hello.com"
}

resource "grafana_cloud_stack" "my-stack" {
  region = data.region.slug
  slug   = "hello"
  other  = ["123", "456"]
  sub_block {
  }
  sub_block {
    attr = "val"
  }
}
