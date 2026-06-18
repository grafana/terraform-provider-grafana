#!/usr/bin/env bash
# Dispatches terraformprovidergrafanatest - deploy on field-eng and waits for completion.
#
# Environment:
#   GH_TOKEN, FIELD_ENG_REPO, BRANCH, ARTIFACT_RUN_ID,
#   FIELD_ENG_DEV_ARTIFACT_NAME, BASE_REF — required (BASE_REF: workflow file ref on field-eng)

set -euo pipefail

: "${GH_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${BRANCH:?}"
: "${ARTIFACT_RUN_ID:?}"
: "${FIELD_ENG_DEV_ARTIFACT_NAME:?}"
: "${BASE_REF:?}"

WORKFLOW_FILE="terraformprovidergrafanatest_deploy.yml"

echo "Dispatching with CI artifact override (run ${ARTIFACT_RUN_ID}, artifact ${FIELD_ENG_DEV_ARTIFACT_NAME})."
dispatch_body="$(jq -n \
  --arg ref "$BASE_REF" \
  --arg deployment_tooling_version "$BRANCH" \
  --arg grafana_provider_dev_override_run_id "$ARTIFACT_RUN_ID" \
  '{
    ref: $ref,
    return_run_details: true,
    inputs: {
      deployment_tooling_version: $deployment_tooling_version,
      grafana_provider_dev_override_run_id: $grafana_provider_dev_override_run_id
    }
  }')"

dispatch_response="$(gh api \
  --method POST \
  -H "Accept: application/vnd.github+json" \
  "repos/${FIELD_ENG_REPO}/actions/workflows/${WORKFLOW_FILE}/dispatches" \
  --input - <<<"$dispatch_body")"

DEPLOY_RUN_ID="$(jq -r '.workflow_run_id // empty' <<<"$dispatch_response")"
RUN_URL="$(jq -r '.html_url // empty' <<<"$dispatch_response")"

if [ -z "$DEPLOY_RUN_ID" ] || [ "$DEPLOY_RUN_ID" = "null" ]; then
  echo "::error::Workflow dispatch did not return workflow_run_id (return_run_details unsupported or empty response)."
  echo "$dispatch_response"
  exit 1
fi

echo "Deploy workflow run URL: ${RUN_URL}"
echo "Deploy workflow run ID: ${DEPLOY_RUN_ID}"

gh run watch "$DEPLOY_RUN_ID" --repo "${FIELD_ENG_REPO}" --exit-status
echo "Deploy workflow completed successfully."
