resource "grafana_mute_timing" "my_mute_timing" {
    name = "My Mute Timing"

    intervals {
        weekdays = ["monday", "tuesday:thursday"]
        days_of_month = ["1:7", "-1"]
    }
}