import {
  provider = grafana.localhost
  to       = grafana_dashboard.localhost_1_my-dashboard-uid
  id       = "1:my-dashboard-uid"
}

import {
  provider = grafana.localhost
  to       = grafana_folder.localhost_1_my-folder-uid
  id       = "1:my-folder-uid"
}

import {
  provider = grafana.localhost
  to       = grafana_notification_policy.localhost_1_policy
  id       = "1:policy"
}
