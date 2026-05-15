#!/usr/bin/env bash
# Dispatches terraformprovidergrafanatest - deploy on field-eng and waits for completion.
#
# Environment:
#   GH_TOKEN, DEPLOYMENT_TOKEN, FIELD_ENG_REPO, BRANCH, DEV_RUN — required
#   FIELD_ENG_DEV_ARTIFACT_NAME — optional; used in log message only (default below)

set -euo pipefail

: "${GH_TOKEN:?}"
: "${DEPLOYMENT_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${BRANCH:?}"
: "${DEV_RUN:?}"

ARTIFACT_NAME="${FIELD_ENG_DEV_ARTIFACT_NAME:-terraform-provider-grafana_linux_amd64}"
START_ISO=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Dispatching with CI artifact override (run ${DEV_RUN}, artifact ${ARTIFACT_NAME})."
gh workflow run "terraformprovidergrafanatest - deploy" \
  --repo "${FIELD_ENG_REPO}" \
  -f "deployment_token=${DEPLOYMENT_TOKEN}" \
  -f "deployment_tooling_version=${BRANCH}" \
  -f "grafana_provider_dev_override_run_id=${DEV_RUN}"

echo "Waiting for new workflow run (started after ${START_ISO})..."
RUN_ID=""
for attempt in $(seq 1 90); do
  sleep 10
  RUN_ID=$(gh run list \
    --repo "${FIELD_ENG_REPO}" \
    --workflow "terraformprovidergrafanatest - deploy" \
    --json databaseId,createdAt,event \
    --jq --arg start "$START_ISO" '
      [.[] | select(.createdAt >= $start and .event == "workflow_dispatch")]
      | sort_by(.createdAt) | reverse | .[0].databaseId // empty
    ')
  if [ -n "$RUN_ID" ] && [ "$RUN_ID" != "null" ]; then
    echo "Found run $RUN_ID"
    break
  fi
  echo "attempt $attempt: run not visible yet..."
done

if [ -z "$RUN_ID" ] || [ "$RUN_ID" = "null" ]; then
  echo "::error::Could not find the dispatched workflow run."
  exit 1
fi

gh run watch "$RUN_ID" --repo "${FIELD_ENG_REPO}" --exit-status
echo "Deploy workflow completed successfully."
