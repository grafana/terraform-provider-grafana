resource "grafana_contact_point" "jira" {
  name = "Jira Contact Point"

  jira {
    api_url             = "https://test.atlassian.net/rest/api/3"
    user                = "test@example.com"
    password            = "test-api-token"
    project             = "TEST"
    issue_type          = "Task"
    summary             = "Alert: {{ .GroupLabels.alertname }}"
    description         = "{{ .Annotations.description }}"
    labels              = ["grafana", "alert"]
    priority            = "High"
    resolve_transition  = "Done"
    reopen_transition   = "To Do"
    reopen_duration     = "10m"
    wont_fix_resolution = "Won't Do"
    dedup_key_field     = "10000"
    fields = {
      customfield_10001 = "custom value"
    }
  }
}
