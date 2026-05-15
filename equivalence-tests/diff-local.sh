#!/usr/bin/env bash
#
# Runs under `make equivalence-test-diff-local` (Makefile sets REPO_ROOT, etc.).
# Builds this repo's provider, writes equivalence-tests/local-provider.tfrc with
# dev_overrides -> REPO_ROOT/testdata/plugins/local-dev, then runs
# terraform-equivalence-testing diff.
#
# Before the diff: prints SHA256 of the built plugin via openssl, echoes
# local-provider.tfrc, and tails terraform init under tests/grafana_team so you
# can confirm the provider development overrides line for grafana/grafana. The
# captured apply.json from the equivalence run also includes the override warning.
#
set -euo pipefail

: "${REPO_ROOT:?REPO_ROOT must be set to the repository root}"

EQUIV_BIN="${EQUIV_BIN:-terraform-equivalence-testing}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_AUTH="${GRAFANA_AUTH:-admin:admin}"

if ! command -v "$EQUIV_BIN" >/dev/null 2>&1; then
	echo "Install the CLI and ensure it is on PATH, or set EQUIV_BIN=/path/to/terraform-equivalence-testing" >&2
	exit 1
fi

LOCAL_PLUGIN="$REPO_ROOT/testdata/plugins/local-dev/terraform-provider-grafana"
TFRC="$REPO_ROOT/equivalence-tests/local-provider.tfrc"
G_TEAM_DIR="$REPO_ROOT/equivalence-tests/tests/grafana_team"

mkdir -p "$REPO_ROOT/testdata/plugins/local-dev"
go build -C "$REPO_ROOT" -o "$LOCAL_PLUGIN" .

cat >"$TFRC" <<EOF
provider_installation {
  dev_overrides {
    "grafana/grafana" = "$REPO_ROOT/testdata/plugins/local-dev"
  }
  direct {}
}
EOF

echo "=== equivalence-test-diff-local: proof (this run uses the binary below + dev_overrides in local-provider.tfrc) ==="
ls -la "$LOCAL_PLUGIN"
echo "SHA256 $(openssl dgst -sha256 "$LOCAL_PLUGIN" | awk '{print $2}')"
echo "--- $TFRC ---"
cat "$TFRC"
echo "--- tail of terraform init in tests/grafana_team (expect Provider development overrides + grafana/grafana + local-dev) ---"
TF_CLI_CONFIG_FILE="$TFRC" GRAFANA_URL="$GRAFANA_URL" GRAFANA_AUTH="$GRAFANA_AUTH" \
	terraform -chdir="$G_TEAM_DIR" init -backend=false -input=false -no-color 2>&1 | tail -n 35

TF_CLI_CONFIG_FILE="$TFRC" GRAFANA_URL="$GRAFANA_URL" GRAFANA_AUTH="$GRAFANA_AUTH" \
	"$EQUIV_BIN" diff \
	--goldens="$REPO_ROOT/equivalence-tests/goldens" \
	--tests="$REPO_ROOT/equivalence-tests/tests"
