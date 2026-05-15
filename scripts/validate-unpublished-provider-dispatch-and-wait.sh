#!/usr/bin/env bash
# Dispatches terraformprovidergrafanatest - deploy on field-eng and waits for completion.
#
# Environment:
#   GH_TOKEN, DEPLOYMENT_TOKEN, FIELD_ENG_REPO, BRANCH, DEV_RUN — required
#   FIELD_ENG_DEV_ARTIFACT_NAME — optional; used in log message only (default below)
#
# Requires GitHub CLI 2.87+ (workflow dispatch returns the created run URL).

set -euo pipefail

: "${GH_TOKEN:?}"
: "${DEPLOYMENT_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${BRANCH:?}"
: "${DEV_RUN:?}"

ARTIFACT_NAME="${FIELD_ENG_DEV_ARTIFACT_NAME:-terraform-provider-grafana_linux_amd64}"
WORKFLOW_NAME="terraformprovidergrafanatest - deploy"

parse_run_id_from_url() {
  local url="$1"
  if [[ "$url" =~ /actions/runs/([0-9]+) ]]; then
    echo "${BASH_REMATCH[1]}"
  fi
}

find_run_url_in_text() {
  local text="$1"
  local line url=""
  while IFS= read -r line; do
    line="${line//$'\r'/}"
    if [[ "$line" =~ ^https://github.com/.*/actions/runs/[0-9]+ ]]; then
      url="$line"
    fi
  done <<<"$text"
  printf '%s' "$url"
}

echo "Dispatching with CI artifact override (run ${DEV_RUN}, artifact ${ARTIFACT_NAME})."
dispatch_output="$(gh workflow run "$WORKFLOW_NAME" \
  --repo "${FIELD_ENG_REPO}" \
  -f "deployment_token=${DEPLOYMENT_TOKEN}" \
  -f "deployment_tooling_version=${BRANCH}" \
  -f "grafana_provider_dev_override_run_id=${DEV_RUN}" \
  2>&1)"

printf '%s\n' "$dispatch_output"

RUN_URL="$(find_run_url_in_text "$dispatch_output")"
if [ -z "$RUN_URL" ]; then
  echo "::error::gh workflow run did not return a run URL. Use GitHub CLI 2.87 or newer (return_run_details)."
  exit 1
fi

RUN_ID="$(parse_run_id_from_url "$RUN_URL")"
if [ -z "$RUN_ID" ]; then
  echo "::error::Could not parse run id from URL: ${RUN_URL}"
  exit 1
fi

echo "Deploy workflow run URL: ${RUN_URL}"
echo "Deploy workflow run ID: ${RUN_ID}"

gh run watch "$RUN_ID" --repo "${FIELD_ENG_REPO}" --exit-status
echo "Deploy workflow completed successfully."
