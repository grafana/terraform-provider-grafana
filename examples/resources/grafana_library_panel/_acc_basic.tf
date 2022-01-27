# The "id" and "version" properties in the config below are there to test that
# we correctly remove them from model_json and manage them in dedicated,
# computed fields.
#
resource "grafana_library_panel" "test" {
  name        = "basic"
  folder_id   = 0
  model_json  = jsonencode({
    title     = "basic",
    type      = "dash-db",
    version   = 34,
  })
}
