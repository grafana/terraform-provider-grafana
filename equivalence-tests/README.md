# Equivalence tests

Uses [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): Terraform runs per `tests/*/spec.json`; **JSON** under `goldens/` (`apply.json`, `plan.json`, `state.json`) is compared to the live run.

## Prerequisites

- `terraform` on `PATH`. Registry `init` needs network.
- Go (to build the repo-local `terraform-equivalence-testing` CLI). The Makefile installs it into `.cache/bin/` on first use; override with `EQUIV_BIN` if needed.
- Defaults when unset: `GRAFANA_URL=http://localhost:3000`, `GRAFANA_AUTH=admin:admin`.

## CLI

```sh
make equivalence-test-install-tool   # optional; update/diff targets install lazily
```

Installs `terraform-equivalence-testing@v0.5.0` into `.cache/bin/` via `go install` (not your global Go bin).

## Commands

```sh
make equivalence-test-update      # refresh goldens/ (registry provider from main.tf)
make equivalence-test-diff        # same provider source as update
make equivalence-test-diff-local  # provider built from this repo vs same goldens
```

Exit `0` = match, `2` = diff, `1` = failed run.

`equivalence-test-update` / `-diff` use the registry Grafana provider with the version pinned per case in `main.tf` and `TF_CLI_CONFIG_FILE` unset. `equivalence-test-diff-local` builds this repo and sets `TF_CLI_CONFIG_FILE` with `dev_overrides` so Terraform loads the local `grafana/grafana` plugin instead of the registry.

If you change the provider version in `main.tf`, refresh `.terraform.lock.hcl` with `terraform init -upgrade` in the relevant test directory to update the provider build used when running equivalence tests.

Some cases use fixed resource names in Grafana. If a run fails because the object already exists (for example HTTP 409), delete the conflicting remote resource before re-running update or diff.

## Adding test cases

Add `tests/<name>/` with `spec.json` and `.tf` files; goldens land in `goldens/<name>/` after `update`.
