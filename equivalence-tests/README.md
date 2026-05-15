# Equivalence tests

Uses [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): Terraform runs per `tests/*/spec.json`; **JSON** under `goldens/` (`apply.json`, `plan.json`, `state.json`) is compared to the live run.

## Prerequisites

- `terraform` and `terraform-equivalence-testing` on `PATH`. Registry `init` needs network. After `make equivalence-test-install-tool` (`go install`), add Go’s bin directory to `PATH`, or set `EQUIV_BIN` to the full path of `terraform-equivalence-testing`.
- Defaults when unset: `GRAFANA_URL=http://localhost:3000`, `GRAFANA_AUTH=admin:admin`.

## CLI

```sh
make equivalence-test-install-tool
```

Uses `go install` (binary in `$GOBIN` or `$(go env GOPATH)/bin`). Put that on `PATH` or set `EQUIV_BIN`.

## Commands

```sh
make equivalence-test-update      # refresh goldens/ (registry provider from main.tf)
make equivalence-test-diff        # same provider source as update
make equivalence-test-diff-local  # provider built from this repo vs same goldens
```

Exit `0` = match, `2` = diff, `1` = failed run.

`equivalence-test-update` / `-diff` use the registry Grafana provider with the version pinned per case in `main.tf` and `TF_CLI_CONFIG_FILE` unset. `equivalence-test-diff-local` builds this repo and sets `TF_CLI_CONFIG_FILE` with `dev_overrides` so Terraform loads the local `grafana/grafana` plugin instead of the registry.

If you change the provider version in `main.tf`, refresh `.terraform.lock.hcl` with `terraform init -upgrade` in the relevant test directory to update the provider build used when running equivalence tests.

The `grafana_team` test uses a fixed team name. If you get a **409** on create: run `make equivalence-test-delete-team` (runs `equivalence-tests/cmd/equiv-delete-team`), then retry.

## Adding test cases

Add `tests/<name>/` with `spec.json` and `.tf` files; goldens land in `goldens/<name>/` after `update`.
