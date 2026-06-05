#!/usr/bin/env bash
#
# Runs under `make equivalence-test-diff-local` (Makefile sets REPO_ROOT and checks the CLI).
# Builds this repo's provider, writes equivalence-tests/local-provider.tfrc with
# dev_overrides -> REPO_ROOT/testdata/plugins/local-dev, then runs
# terraform-equivalence-testing diff.
#
# Before the diff: prints SHA256 of the built plugin via openssl and echoes
# local-provider.tfrc so you can confirm which binary and dev_overrides apply.
#
set -euo pipefail

: "${REPO_ROOT:?REPO_ROOT must be set to the repository root}"

EQUIV_BIN="${EQUIV_BIN:-terraform-equivalence-testing}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_AUTH="${GRAFANA_AUTH:-admin:admin}"

LOCAL_PLUGIN="$REPO_ROOT/testdata/plugins/local-dev/terraform-provider-grafana"
TFRC="$REPO_ROOT/equivalence-tests/local-provider.tfrc"

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

TF_CLI_CONFIG_FILE="$TFRC" GRAFANA_URL="$GRAFANA_URL" GRAFANA_AUTH="$GRAFANA_AUTH" \
	"$EQUIV_BIN" diff \
	--goldens="$REPO_ROOT/equivalence-tests/goldens" \
	--tests="$REPO_ROOT/equivalence-tests/tests"
