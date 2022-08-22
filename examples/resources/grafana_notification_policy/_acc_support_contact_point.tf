resource "grafana_mute_timing" "a_mute_timing" {
    name = "Some Mute Timing"

    intervals {
        weekdays = ["monday"]
    }
}
