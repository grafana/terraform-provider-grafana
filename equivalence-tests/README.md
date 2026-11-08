# Equivalence tests

Uses [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): Terraform runs per `tests/*/spec.json`; **JSON** under `goldens/` (`apply.json`, `plan.json`, `state.json`) is compared to the live run.

## Prerequisites

- `terraform` on `PATH`. Registry `init` needs network.
- Go (to build the repo-local `terraform-equivalence-testing` CLI). The Makefile installs it into `.cache/bin/` on first `update`/`diff` run; override with `EQUIV_BIN` if needed.
- Docker (starts a fresh Grafana via `docker compose` for each command below).

## Commands

Each target starts a fresh Grafana (same stack as `make testacc-oss-docker`), runs the test, then tears down compose.

```sh
make equivalence-test-update      # refresh goldens/ (registry provider from main.tf)
make equivalence-test-diff        # same provider source as update
make equivalence-test-diff-local  # provider built from this repo vs same goldens
```

Run a single case (or subset) with `EQUIV_FILTERS` — comma-separated names matching directories under `tests/`:

```sh
make equivalence-test-diff-local EQUIV_FILTERS=grafana_user
make equivalence-test-diff EQUIV_FILTERS=grafana_user
make equivalence-test-update EQUIV_FILTERS=grafana_user,grafana_team
```

Exit `0` = match, `2` = diff, `1` = failed run.

`equivalence-test-update` / `-diff` use the registry Grafana provider with the version pinned per case in `main.tf` and `TF_CLI_CONFIG_FILE` unset. `equivalence-test-diff-local` builds this repo and sets `TF_CLI_CONFIG_FILE` with `dev_overrides` so Terraform loads the local `grafana/grafana` plugin instead of the registry.

If you change the provider version in `main.tf`, refresh `.terraform.lock.hcl` with `terraform init -upgrade` in the relevant test directory to update the provider build used when running equivalence tests.

## Adding test cases

Add `tests/<name>/` with `spec.json` and `.tf` files; goldens land in `goldens/<name>/` after `update`.
