resource "grafana_assistant_quickstart" "example" {
  scope  = "tenant"
  title  = "SLO health"
  prompt = "How healthy are my SLOs right now?"
}

# A quickstart with a context item pre-attached. `context_items` is a
# JSON-encoded array of Assistant `ChatContextItem` objects. The example below
# attaches a Prometheus data source so the Assistant starts the conversation
# with that data source already in context.
#
# This is an advanced, internal-format field. The most reliable way to obtain a
# valid value is to create the quickstart with the desired context through the
# Assistant UI and copy the resulting `contextItems` JSON into `jsonencode(...)`.
resource "grafana_assistant_quickstart" "with_context" {
  scope  = "tenant"
  title  = "Investigate Prometheus alerts"
  prompt = "Which alerts are firing right now and why?"

  context_items = jsonencode([
    {
      node = {
        id   = "prometheus-uid"
        name = "Prometheus"
        icon = "database"
        data = {
          type = "datasource"
          data = {
            name = "Prometheus"
            uid  = "prometheus-uid"
            type = "prometheus"
            text = "Prometheus"
          }
        }
      }
    }
  ])
}
