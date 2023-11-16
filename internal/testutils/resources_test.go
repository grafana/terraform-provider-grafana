package testutils

import "testing"

func TestWithoutResource(t *testing.T) {
	t.Parallel()

	input := `resource "grafana_organization" "test" {
		name = "test"
	}
	
	resource "grafana_folder" "test" {
		org_id = grafana_organization.test.id
		title = "test"
	}
	
	resource "grafana_rule_group" "test" {
		org_id = grafana_organization.test.id
		name             = "test"
		folder_uid       = grafana_folder.test.uid
		interval_seconds = 360
		rule {
			name           = "My Alert Rule 1"
			for            = "2m"
			condition      = "B"
			no_data_state  = "NoData"
			exec_err_state = "Alerting"
			is_paused = false
			data {
				ref_id     = "A"
				query_type = ""
				relative_time_range {
					from = 600
					to   = 0
				}
				datasource_uid = "PD8C576611E62080A"
				model = jsonencode({
					hide          = false
					intervalMs    = 1000
					maxDataPoints = 43200
					refId         = "A"
				})
			}
		}
	}`

	expected := `resource "grafana_organization" "test" {
  name = "test"
}

resource "grafana_folder" "test" {
  org_id = grafana_organization.test.id
  title  = "test"
}

`

	actual := WithoutResource(t, input, "grafana_rule_group.test")

	if actual != expected {
		t.Errorf("expected:\n%q\n\nactual:\n%q", expected, actual)
	}
}
