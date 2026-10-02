#!/usr/bin/env bash
#
# Runs a Makefile equivalence-test-*-run target against a fresh Grafana from docker compose.
# Starts compose with --force-recreate and --renew-anon-volumes so each run gets empty
# MySQL/Grafana state; tears down on exit (including test failure).
#
# Usage: run-with-grafana.sh <make-target>
#
set -euo pipefail

if [[ $# -lt 1 ]]; then
	echo "usage: $0 <make-target>" >&2
	exit 1
fi

REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
GRAFANA_URL="${GRAFANA_URL:-http://0.0.0.0:3000}"
GRAFANA_AUTH="${GRAFANA_AUTH:-admin:admin}"
GRAFANA_VERSION="${GRAFANA_VERSION:-latest}"
EQUIV_FILTERS="${EQUIV_FILTERS:-}"
DOCKER_COMPOSE_ARGS="${DOCKER_COMPOSE_ARGS:---pull always --force-recreate --detach --remove-orphans --wait --renew-anon-volumes}"

cleanup() {
	docker compose -f "$REPO_ROOT/docker-compose.yml" down
}
trap cleanup EXIT

cd "$REPO_ROOT"
export GRAFANA_URL GRAFANA_AUTH GRAFANA_VERSION

IFS=' ' read -r -a compose_args <<<"$DOCKER_COMPOSE_ARGS"
docker compose up "${compose_args[@]}"

status=0
make -C "$REPO_ROOT" EQUIV_FILTERS="$EQUIV_FILTERS" "$1" || status=$?
exit "$status"
