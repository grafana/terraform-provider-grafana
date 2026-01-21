# Test adding a uid to a rule that previously didn't have one
resource "grafana_folder" "test_folder" {
  title = "Test Folder"
  uid   = "test-folder-unset"
}

resource "grafana_alerting_rule" "test" {
  config_json = jsonencode({
    title        = "UID Unset"
    uid          = "uid-previously-unset"
    condition    = "B"
    folderUID    = grafana_folder.test_folder.uid
    ruleGroup    = "Test Group"
    noDataState  = "NoData"
    execErrState = "Error"
    for          = "0m"
    data = [
      {
        refId         = "B"
        datasourceUid = "__expr__"
        model = {
          refId = "B"
          type  = "classic_conditions"
          conditions = [
            {
              type     = "query"
              operator = { type = "and" }
              query    = { params = ["A"] }
              evaluator = {
                type   = "gt"
                params = [0]
              }
            }
          ]
          datasource = {
            type = "__expr__"
            uid  = "__expr__"
          }
          expression = "A"
          reducer    = "last"
        }
        relativeTimeRange = {
          from = 600
          to   = 0
        }
      }
    ]
  })
}
