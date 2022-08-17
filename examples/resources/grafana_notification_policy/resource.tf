resource "grafana_contact_point" "a_contact_point" {
    name = "A Contact Point"

   email {
        addresses = ["one@company.org", "two@company.org"]
        message = "{{ len .Alerts.Firing }} firing."
    }
}


resource "grafana_notification_policy" "my_notification_policy" {
    group_by = ["..."]
    contact_point = grafana_contact_point.a_contact_point.name

    group_wait = "45s"
    group_interval = "6m"
    repeat_interval = "3h"
}
