module github.com/grafana/terraform-provider-grafana

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/grafana/grafana-api-golang-client v0.3.1
	github.com/grafana/machine-learning-go-client v0.1.1
	github.com/grafana/synthetic-monitoring-agent v0.6.2
	github.com/grafana/synthetic-monitoring-api-go-client v0.5.1
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/terraform-plugin-docs v0.5.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.10.1
)

// replace github.com/grafana/grafana-api-golang-client v0.3.0 => /Users/justin.mai/git/grafana-api-golang-client-1
replace github.com/grafana/grafana-api-golang-client v0.3.0 => github.com/justinTM/grafana-api-golang-client v0.2.2-0.20220208043510-6589b07f383b
