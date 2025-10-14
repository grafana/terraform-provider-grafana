terraform import grafana_dashboard_public.name "{{ dashboardUID }}:{{ publicDashboardUID }}"
terraform import grafana_dashboard_public.name "{{ orgID }}:{{ dashboardUID }}:{{ publicDashboardUID }}"
