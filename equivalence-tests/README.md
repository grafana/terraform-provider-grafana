# Equivalence tests

Uses [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): Terraform runs per `tests/*/spec.json`; **JSON** under `goldens/` (`apply.json`, `plan.json`, `state.json`) is compared to the live run.

## Prerequisites

- `terraform` on `PATH`; network for registry `init`
- `terraform-equivalence-testing` on `PATH`, or `make equivalence-test-install-tool` (needs Go)
- Grafana reachable. `make equivalence-test-update` / `diff` default `GRAFANA_URL` to `http://localhost:3000` and `GRAFANA_AUTH` to `admin:admin` if unset, and unset `TF_CLI_CONFIG_FILE` (provider from registry per `tests/grafana_team/main.tf`).

## CLI

```sh
make equivalence-test-install-tool
```

Use `EQUIV_BIN` if the binary is not on `PATH`.

## Commands

```sh
make equivalence-test-update   # refresh goldens/
make equivalence-test-diff     # compare live run to goldens/
```

Exit `0` = match, `2` = diff, `1` = failed run.

If you change `required_providers` in `main.tf`, refresh `.terraform.lock.hcl` with `terraform init -upgrade` in `tests/grafana_team/` before relying on a pinned install.

The `grafana_team` test uses a fixed team name; **409** on create: `make equivalence-test-delete-team`, then retry.

## Cases

| Test directory | Resource |
|----------------|----------|
| `tests/grafana_team/` | `grafana_team` |

Add `tests/<name>/` with `spec.json` and `.tf` files; goldens land in `goldens/<name>/` after `update`.
