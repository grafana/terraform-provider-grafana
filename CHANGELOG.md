# Changelog

## [v1.6.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.6.0) (2020-10-16)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.5.0...v1.6.0)

**Implemented enhancements:**

- grafana\_data\_source does not set the service account key for the stackdriver datasource [\#91](https://github.com/grafana/terraform-provider-grafana/issues/91)
- Setting the alert notification channel uid [\#83](https://github.com/grafana/terraform-provider-grafana/issues/83)
- \[Feature Request\] Support setting version for Elasticsearch datasource [\#54](https://github.com/grafana/terraform-provider-grafana/issues/54)
- Allow skipping TLS verify in datasources [\#42](https://github.com/grafana/terraform-provider-grafana/issues/42)
- Targets/Docs for Releases and Changelog Generation [\#127](https://github.com/grafana/terraform-provider-grafana/pull/127) ([trotttrotttrott](https://github.com/trotttrotttrott))
- grafana\_user is importable [\#125](https://github.com/grafana/terraform-provider-grafana/pull/125) ([trotttrotttrott](https://github.com/trotttrotttrott))
- Automated Releases [\#123](https://github.com/grafana/terraform-provider-grafana/pull/123) ([trotttrotttrott](https://github.com/trotttrotttrott))
- resource/data\_source: add support for stackdriver privatekey [\#100](https://github.com/grafana/terraform-provider-grafana/pull/100) ([anGie44](https://github.com/anGie44))
- Add uid value to alert notification resource [\#87](https://github.com/grafana/terraform-provider-grafana/pull/87) ([58231](https://github.com/58231))

**Closed issues:**

- We should be able to inject our own UID with notification channel and dashboards [\#115](https://github.com/grafana/terraform-provider-grafana/issues/115)
- New upstream release, can we get a new provider release? [\#97](https://github.com/grafana/terraform-provider-grafana/issues/97)
- Data Source resource docs are missing information about access\_mode. [\#92](https://github.com/grafana/terraform-provider-grafana/issues/92)
- Would it help to use a more complete version of Go SDK? [\#71](https://github.com/grafana/terraform-provider-grafana/issues/71)
- Prompted for 'auth' on terraform plan execution [\#66](https://github.com/grafana/terraform-provider-grafana/issues/66)
- Invalid CA [\#53](https://github.com/grafana/terraform-provider-grafana/issues/53)
- Resources to configure grafana server \(installation\) configuration? [\#48](https://github.com/grafana/terraform-provider-grafana/issues/48)
- \[Feature Request\] Import existing dashboards into terraform [\#24](https://github.com/grafana/terraform-provider-grafana/issues/24)
- expand json\_data field usage. [\#22](https://github.com/grafana/terraform-provider-grafana/issues/22)

**Merged pull requests:**

- Examples are formatted properly [\#126](https://github.com/grafana/terraform-provider-grafana/pull/126) ([trotttrotttrott](https://github.com/trotttrotttrott))
- Update From Grafana Fork [\#122](https://github.com/grafana/terraform-provider-grafana/pull/122) ([trotttrotttrott](https://github.com/trotttrotttrott))
- Adding team resource functionality [\#120](https://github.com/grafana/terraform-provider-grafana/pull/120) ([jonathan-dorsey](https://github.com/jonathan-dorsey))
- Settings is an argument not a block [\#114](https://github.com/grafana/terraform-provider-grafana/pull/114) ([Arola1982](https://github.com/Arola1982))
- Update link to documentation [\#99](https://github.com/grafana/terraform-provider-grafana/pull/99) ([tonglil](https://github.com/tonglil))
- Fix build, use -mod=readonly [\#98](https://github.com/grafana/terraform-provider-grafana/pull/98) ([tonglil](https://github.com/tonglil))
- Allow alert notification reminder to be turned on [\#94](https://github.com/grafana/terraform-provider-grafana/pull/94) ([jvshahid](https://github.com/jvshahid))
- Updating the access\_mode setting description. [\#93](https://github.com/grafana/terraform-provider-grafana/pull/93) ([phillipsj](https://github.com/phillipsj))
- Update resource grafana\_data\_source [\#90](https://github.com/grafana/terraform-provider-grafana/pull/90) ([mlclmj](https://github.com/mlclmj))
- Document 'folder' attribute [\#86](https://github.com/grafana/terraform-provider-grafana/pull/86) ([jeohist](https://github.com/jeohist))
- Mark secret\_key in secure\_json\_data as sensitive [\#78](https://github.com/grafana/terraform-provider-grafana/pull/78) ([Infra-Red](https://github.com/Infra-Red))
- deps: Bump nytm/go-grafana-api to 0.2.0 [\#75](https://github.com/grafana/terraform-provider-grafana/pull/75) ([radeksimko](https://github.com/radeksimko))
- Argument names must not be quoted [\#73](https://github.com/grafana/terraform-provider-grafana/pull/73) ([tomweston](https://github.com/tomweston))
- Provider logging [\#46](https://github.com/grafana/terraform-provider-grafana/pull/46) ([radeksimko](https://github.com/radeksimko))
- Update slack alert notification example usage [\#45](https://github.com/grafana/terraform-provider-grafana/pull/45) ([alex-stiff](https://github.com/alex-stiff))

## [v1.5.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.5.0) (2019-06-26)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.4.0...v1.5.0)

**Closed issues:**

- ReadDataSource fails if the data source is not there [\#55](https://github.com/grafana/terraform-provider-grafana/issues/55)

**Merged pull requests:**

- Check for data source 404 in the correct place [\#56](https://github.com/grafana/terraform-provider-grafana/pull/56) ([sjauld](https://github.com/sjauld))
- Update dashboards with correct ForceNew on folder [\#52](https://github.com/grafana/terraform-provider-grafana/pull/52) ([ghmeier](https://github.com/ghmeier))

## [v1.4.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.4.0) (2019-05-22)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.3.0...v1.4.0)

**Closed issues:**

- Documentation missing quote [\#39](https://github.com/grafana/terraform-provider-grafana/issues/39)

**Merged pull requests:**

- Update to TF SDK v0.12 [\#61](https://github.com/grafana/terraform-provider-grafana/pull/61) ([paultyng](https://github.com/paultyng))
- switch to modules and vendor 0.12 sdk [\#44](https://github.com/grafana/terraform-provider-grafana/pull/44) ([appilon](https://github.com/appilon))
- \[AUTOMATED\] Upgrade to Go 1.11 [\#41](https://github.com/grafana/terraform-provider-grafana/pull/41) ([appilon](https://github.com/appilon))

## [v1.3.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.3.0) (2018-11-16)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.2.0...v1.3.0)

**Implemented enhancements:**

- Import error debug [\#30](https://github.com/grafana/terraform-provider-grafana/pull/30) ([tonglil](https://github.com/tonglil))

**Closed issues:**

- PagerDuty setting for grafana\_alert\_notification is coerced into an invalid value [\#35](https://github.com/grafana/terraform-provider-grafana/issues/35)
- POSTing to Comodo-certified grafana URL fails with x509: certificate signed by unknown authority [\#34](https://github.com/grafana/terraform-provider-grafana/issues/34)

**Merged pull requests:**

- support boolean settings for alert notifications [\#37](https://github.com/grafana/terraform-provider-grafana/pull/37) ([DanCech](https://github.com/DanCech))
- Add support for creating folders and creating dashboards inside folders [\#36](https://github.com/grafana/terraform-provider-grafana/pull/36) ([goraxe](https://github.com/goraxe))
- Add missing quotes in grafana\_organization docs [\#32](https://github.com/grafana/terraform-provider-grafana/pull/32) ([illagrenan](https://github.com/illagrenan))

## [v1.2.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.2.0) (2018-08-01)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.1.0...v1.2.0)

**Merged pull requests:**

- Resource Organization [\#29](https://github.com/grafana/terraform-provider-grafana/pull/29) ([mlclmj](https://github.com/mlclmj))

## [v1.1.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.1.0) (2018-07-27)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.0.2...v1.1.0)

**Closed issues:**

- Upstream Library Ownership [\#26](https://github.com/grafana/terraform-provider-grafana/issues/26)

**Merged pull requests:**

- fix\(Schema\): Mark arguments containing secrets as sensitive [\#28](https://github.com/grafana/terraform-provider-grafana/pull/28) ([donoftime](https://github.com/donoftime))
- Change of Library [\#27](https://github.com/grafana/terraform-provider-grafana/pull/27) ([mlclmj](https://github.com/mlclmj))
- make: Add website + website-test targets [\#21](https://github.com/grafana/terraform-provider-grafana/pull/21) ([radeksimko](https://github.com/radeksimko))

## [v1.0.2](https://github.com/grafana/terraform-provider-grafana/tree/v1.0.2) (2018-04-18)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.0.1...v1.0.2)

**Implemented enhancements:**

- alert\_notification/dashboard: fix compatibility with grafana 5.0 [\#17](https://github.com/grafana/terraform-provider-grafana/pull/17) ([pearkes](https://github.com/pearkes))

**Closed issues:**

- Grafana 5.0 Dashboard Support [\#15](https://github.com/grafana/terraform-provider-grafana/issues/15)
- Grafana Datasource Cloudwatch ARN missing attributes [\#14](https://github.com/grafana/terraform-provider-grafana/issues/14)
- Make url field optional for grafana\_data\_source to support Cloudwatch [\#13](https://github.com/grafana/terraform-provider-grafana/issues/13)
- Document and support non-InfluxDB datasources [\#4](https://github.com/grafana/terraform-provider-grafana/issues/4)

**Merged pull requests:**

- Update readme and add a shortcut to running grafana locally [\#20](https://github.com/grafana/terraform-provider-grafana/pull/20) ([pearkes](https://github.com/pearkes))
- data\_source: make URL field optional [\#18](https://github.com/grafana/terraform-provider-grafana/pull/18) ([pearkes](https://github.com/pearkes))

## [v1.0.1](https://github.com/grafana/terraform-provider-grafana/tree/v1.0.1) (2018-01-12)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v1.0.0...v1.0.1)

**Implemented enhancements:**

- Handle 404 response on Read [\#12](https://github.com/grafana/terraform-provider-grafana/pull/12) ([sl1pm4t](https://github.com/sl1pm4t))

**Merged pull requests:**

- Updated vendored go-grafana-api client. [\#9](https://github.com/grafana/terraform-provider-grafana/pull/9) ([sl1pm4t](https://github.com/sl1pm4t))

## [v1.0.0](https://github.com/grafana/terraform-provider-grafana/tree/v1.0.0) (2017-10-23)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/v0.1.0...v1.0.0)

**Implemented enhancements:**

- Be nicer when a dashboard is deleted from grafana [\#7](https://github.com/grafana/terraform-provider-grafana/pull/7) ([roidelapluie](https://github.com/roidelapluie))
- AWS cloudwatch data source support [\#5](https://github.com/grafana/terraform-provider-grafana/pull/5) ([mdb](https://github.com/mdb))
- Implemented alert\_notification management [\#3](https://github.com/grafana/terraform-provider-grafana/pull/3) ([mvisonneau](https://github.com/mvisonneau))

**Closed issues:**

- Separator between username and password is not mentioned in documentation [\#1](https://github.com/grafana/terraform-provider-grafana/issues/1)

**Merged pull requests:**

- Fix data source config [\#6](https://github.com/grafana/terraform-provider-grafana/pull/6) ([roidelapluie](https://github.com/roidelapluie))
- vendor: github.com/hashicorp/terraform/...@v0.10.0 [\#2](https://github.com/grafana/terraform-provider-grafana/pull/2) ([radeksimko](https://github.com/radeksimko))

## [v0.1.0](https://github.com/grafana/terraform-provider-grafana/tree/v0.1.0) (2017-06-20)

[Full Changelog](https://github.com/grafana/terraform-provider-grafana/compare/6e45b80f7bbe6f449a4641a3f32213a9226d7830...v0.1.0)



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
