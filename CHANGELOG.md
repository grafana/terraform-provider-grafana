## 1.6.0 (Unreleased)

IMPROVEMENTS:

* provider: Improve provider logging [GH-46]
* resource/alert_notification: Add `uid` field [GH-87]
* resource/alert_notification: Add `send_reminder` and `frequency` fields [GH-94]
* resource/dashboard: Document `folder` field [GH-86]
* resource/data_source: Support additional data sources [GH-90]
* resource/data_source: Update the `access_mode` field description [GH-93]

BUG FIXES:

* resource/data_source: Mark `secret_key` in `secure_json_data` as sensitive [GH-78]
* resource/alert_notification: Update example fields [GH-45]
* resource/alert_notification: Update example formatting [GH-73]

## 1.5.0 (June 26, 2019)

IMPROVEMENTS

* `grafana_dashboard` - Add update support ([#52](https://github.com/terraform-providers/terraform-provider-grafana/issues/52))

BUG FIXES:

* `grafana_data_source` - Fix 404 check ([#56](https://github.com/terraform-providers/terraform-provider-grafana/issues/56))

## 1.4.0 (May 22, 2019)

IMPROVEMENTS:

The provider is now compatible with Terraform v0.12, while retaining compatibility with prior versions.

## 1.3.0 (November 16, 2018)

FEATURES:

* **New Resource:** `grafana_folder` ([#36](https://github.com/terraform-providers/terraform-provider-grafana/issues/36))

IMPROVEMENTS:

* `grafana_dashboard` - Add support for creating dashboards inside folders ([#36](https://github.com/terraform-providers/terraform-provider-grafana/issues/36))
* `grafana_organization` - Add missing quotes in docs ([#32](https://github.com/terraform-providers/terraform-provider-grafana/issues/32))
* `grafana_organization` - Better import error debugging ([#30](https://github.com/terraform-providers/terraform-provider-grafana/issues/30))

BUG FIXES:

* `grafana_alert_notification` - Support boolean settings for alert notifications ([#37](https://github.com/terraform-providers/terraform-provider-grafana/issues/37))

## 1.2.0 (August 01, 2018)

FEATURES:
* **New Resource:** `grafana_resource_organization` ([#29](https://github.com/terraform-providers/terraform-provider-grafana/issues/29))

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
