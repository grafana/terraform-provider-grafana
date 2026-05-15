#!/usr/bin/env bash
# Resolves workflow_dispatch inputs and writes validate job configuration to GITHUB_OUTPUT.
#
# Environment:
#   INPUT_REPO, INPUT_BASE, INPUT_DELETE — from workflow_dispatch inputs
#   GITHUB_OUTPUT — required

set -euo pipefail

REPO="${INPUT_REPO:-grafana/field-eng-appenv-deployment}"
BASE="${INPUT_BASE:-main}"
DELETE_AFTER="${INPUT_DELETE:-true}"

{
  echo "field_eng_repo=$REPO"
  echo "base_ref=$BASE"
  echo "delete_after=$DELETE_AFTER"
} >>"$GITHUB_OUTPUT"

echo "Resolved: repo=$REPO base=$BASE delete_after=$DELETE_AFTER dev_override_run_id=${GITHUB_RUN_ID:-unknown}"
