# Equivalence tests

Uses [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): Terraform runs per `tests/*/spec.json`; **JSON** under `goldens/` (`apply.json`, `plan.json`, `state.json`) is compared to the live run.

## Prerequisites

- `terraform` on `PATH`; network for registry `init`
- `terraform-equivalence-testing` on `PATH`, or `make equivalence-test-install-tool` (needs Go)
- Grafana reachable. `make equivalence-test-update` / `equivalence-test-diff` default `GRAFANA_URL` / `GRAFANA_AUTH` and unset `TF_CLI_CONFIG_FILE` (registry provider per `tests/grafana_team/main.tf`). `make equivalence-test-diff-local` builds this repo’s provider and uses `TF_CLI_CONFIG_FILE` + `dev_overrides` for `grafana/grafana` instead.

## CLI

```sh
make equivalence-test-install-tool
```

Use `EQUIV_BIN` if the binary is not on `PATH`.

## Commands

```sh
make equivalence-test-update      # refresh goldens/ (registry provider from main.tf)
make equivalence-test-diff        # same provider source as update
make equivalence-test-diff-local  # provider built from this repo vs same goldens
```

Exit `0` = match, `2` = diff, `1` = failed run.

`equivalence-test-diff-local` prints **SHA256** of the built plugin, the generated **`local-provider.tfrc`** (`dev_overrides` → `testdata/plugins/local-dev`), and the **tail of `terraform init`** so you can see Terraform’s **Provider development overrides** line naming `grafana/grafana` and that directory. During the diff, **`apply.json`** also includes the same override warning text.

If you change `required_providers` in `main.tf`, refresh `.terraform.lock.hcl` with `terraform init -upgrade` in `tests/grafana_team/` before relying on a pinned install.

The `grafana_team` test uses a fixed team name; **409** on create: `make equivalence-test-delete-team`, then retry.

## Cases

| Test directory | Resource |
|----------------|----------|
| `tests/grafana_team/` | `grafana_team` |

Add `tests/<name>/` with `spec.json` and `.tf` files; goldens land in `goldens/<name>/` after `update`.
