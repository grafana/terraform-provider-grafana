## 1.2.0 (Unreleased)

FEATURES:
* **New Resource:** `grafana_resource_organization` [GH-29]

## 1.1.0 (July 27, 2018)

IMPROVEMENTS:

* `provider` - mark various secret fields as sensitive ([#28](https://github.com/terraform-providers/terraform-provider-grafana/issues/28))

## 1.0.2 (April 18, 2018)

IMPROVEMENTS:

* `grafana_data_source` - make the url field optional to support resources like cloudwatch ([#18](https://github.com/terraform-providers/terraform-provider-grafana/pull/18))
* `alert_notification` - handle null response in grafana 5.0 for non-existent dashboards ([#17](https://github.com/terraform-providers/terraform-provider-grafana/pull/17))
* `grafana_data_source` - additional support for cloudwatch options (`custom_metrics_namespaces`, `assume_role_arn`) ([#16](https://github.com/terraform-providers/terraform-provider-grafana/pull/16))

## 1.0.1 (January 12, 2018)

IMPROVEMENTS:

* `grafana_alert_notification` - handle resources deleted out of band ([#12](https://github.com/terraform-providers/terraform-provider-grafana/issues/12))
* `grafana_data_source` - handle resources deleted out of band ([#12](https://github.com/terraform-providers/terraform-provider-grafana/issues/12))

## 1.0.0 (October 23, 2017)

FEATURES:

* **New Resource:** `alert_notification` ([#3](https://github.com/terraform-providers/terraform-provider-grafana/issues/3))

IMPROVEMENTS:

* resource/grafana_dashboard: Be nicer when a dashboard is deleted ([#7](https://github.com/terraform-providers/terraform-provider-grafana/issues/7))
* resource/grafana_data_source: Support `json_data` and `secure_json_data` arguments to support more data sources, including AWS CloudWatch ([#5](https://github.com/terraform-providers/terraform-provider-grafana/issues/5))

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
