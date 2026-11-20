# Integration schema snapshot

`integration-schemas.json` describes the integrations that Grafana Alerting can create (Slack, email, webhook and
so on) and the settings of each: names, kinds, required and secure fields, allowed values and descriptions. It is
embedded into the provider (`go:embed`), so no schemas are fetched from Grafana at runtime.

It is generated, so do not edit it by hand. It is a reduced copy of the list response of Grafana's
`IntegrationTypeSchema` API: only the integration types and versions that can be created, and only what this
package uses of each field. The Terraform-specific parts (names, types, sensitivity) are not in it: `Load()` applies
them when the snapshot is loaded, and turns it into `IntegrationType` and `Field` values described in Terraform
terms (`model.go`). They belong in `overrides.go`, keyed by integration type, version and field path.

## Provenance

The `source` at the top of the file records where it comes from: the commit of `grafana/grafana`, and the file in
it, which is `pkg/tests/apis/alerting/notifications/integrationtypeschema/test-data/list.json`. Grafana's own test
rewrites that file when the API response changes, so it matches the API at that commit.

## Updating

Pick a commit of `grafana/grafana` (usually the latest on `main`) and run:

```sh
make update-integration-schemas GRAFANA_SHA=<full commit SHA>
```

It downloads the file at that commit, reduces it, and rewrites `integration-schemas.json`. Review the diff (one
line per field), then run `go test ./internal/resources/appplatform/...`. The receiver schema snapshot test
(`receiver_schema_snapshot_test.go`) rewrites `testdata/receiver_schema.golden` and fails when the schema changed:
review that diff, run the tests again, and run `make docs`. To read the file from disk instead of
GitHub, for example for a Grafana branch that is not pushed yet, run the command directly:

```sh
go run ./internal/resources/appplatform/integrationschemas/cmd/update -sha <commit SHA> -file <path to list.json>
```

The command fails on an unexpected `apiVersion` or `kind`, keys that the API client's types don't know, and field
kinds it cannot represent, instead of dropping them. That usually means the
`github.com/grafana/grafana/apps/alerting/notifications` dependency needs updating before the snapshot can be.
