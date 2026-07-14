#!/usr/bin/env bash
# Dispatches field-eng "Deploy AppEnv" (generic_deploy.yml) for tfprovidertest and waits for completion.
#
# Environment:
#   GH_TOKEN, FIELD_ENG_REPO, ARTIFACT_RUN_ID, BASE_REF — required (BASE_REF: git ref on field-eng)

set -euo pipefail

: "${GH_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${ARTIFACT_RUN_ID:?}"
: "${BASE_REF:?}"

WORKFLOW_FILE="generic_deploy.yml"
DEPLOYMENT_CONFIG="tfprovidertest"

echo "Dispatching Deploy AppEnv (${DEPLOYMENT_CONFIG}) with CI artifact override (run ${ARTIFACT_RUN_ID})."
dispatch_body="$(jq -n \
  --arg ref "$BASE_REF" \
  --arg deployment_config "$DEPLOYMENT_CONFIG" \
  --arg grafana_provider_dev_override_run_id "$ARTIFACT_RUN_ID" \
  '{
    ref: $ref,
    return_run_details: true,
    inputs: {
      deployment_config: $deployment_config,
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
