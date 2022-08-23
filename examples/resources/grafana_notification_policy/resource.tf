resource "grafana_contact_point" "a_contact_point" {
    name = "A Contact Point"

    email {
        addresses = ["one@company.org", "two@company.org"]
        message = "{{ len .Alerts.Firing }} firing."
    }
}

resource "grafana_mute_timing" "a_mute_timing" {
    name = "Some Mute Timing"

    intervals {
        weekdays = ["monday"]
    }

    depends_on = [
        grafana_contact_point.a_contact_point
    ]
}


resource "grafana_notification_policy" "my_notification_policy" {
    group_by = ["..."]
    contact_point = grafana_contact_point.a_contact_point.name

    group_wait = "45s"
    group_interval = "6m"
    repeat_interval = "3h"

    policy {
        matcher {
            label = "mylabel"
            match = "="
            value = "myvalue"
        }
        contact_point = grafana_contact_point.a_contact_point.name
        group_by = ["alertname"]
        continue = true
        mute_timings = [grafana_mute_timing.a_mute_timing.name]

        group_wait = "45s"
        group_interval = "6m"
        repeat_interval = "3h"

        policy {
            matcher {
                label = "sublabel"
                match = "="
                value = "subvalue"
            }
            contact_point = grafana_contact_point.a_contact_point.name
            group_by = ["..."]
        }
    }

     policy {
        matcher {
            label = "anotherlabel"
            match = "=~"
            value = "another value.*"
        }
        contact_point = grafana_contact_point.a_contact_point.name
        group_by = ["..."]
    }
}
