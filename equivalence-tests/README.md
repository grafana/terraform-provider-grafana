# Equivalence tests

These tests use [terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing): they run `terraform init` / `plan` / `apply`, capture plan and state JSON, and compare against checked-in **goldens** under `goldens/`.

## Prerequisites

- `terraform` on `PATH`
- Go (to build the provider and install the equivalence CLI)
- A reachable Grafana instance; configure the provider with env vars (same idea as acceptance tests), for example:
  - `GRAFANA_URL` (e.g. `http://localhost:3000`)
  - `GRAFANA_AUTH` (defaults to `admin:admin` in the Makefile if unset)

## One-time CLI install

```sh
make equivalence-test-install-tool
```

This installs `terraform-equivalence-testing` into `$(go env GOPATH)/bin`. Override the binary path with `EQUIV_BIN` if you install it elsewhere.

## Updating goldens

After changing the provider in a way that should intentionally change Terraform output, refresh the golden files:

```sh
export GRAFANA_URL=http://localhost:3000   # your instance
make equivalence-test-update
```

Then review and commit changes under `equivalence-tests/goldens/`.

The `grafana_team` case uses a **fixed** team name (`terraform-equivalence-grafana-team`). Each run uses a new temporary working directory, so Terraform always tries to **create** that team again. If it already exists in Grafana, delete it first (or use a clean org), then re-run.

## Checking for unexpected drift

```sh
export GRAFANA_URL=http://localhost:3000
make equivalence-test-diff
```

Exit code `0` means no differences; `2` means the live output differed from goldens (see the tool’s logs). `1` indicates a failed run (e.g. Terraform or provider error).

## Which tests exist

| Case directory              | Resource    |
|----------------------------|-------------|
| `tests/grafana_team/`      | `grafana_team` |

Add new directories under `tests/<name>/` with `spec.json` plus `.tf` files; goldens land in `goldens/<name>/` after `update`.
