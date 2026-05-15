#!/usr/bin/env bash
# Resolves workflow_dispatch inputs and writes validate job configuration to GITHUB_OUTPUT.
#
# Environment:
#   INPUT_VERSION, INPUT_REPO, INPUT_BASE, INPUT_DELETE — from workflow_dispatch inputs
#   GITHUB_OUTPUT — required

set -euo pipefail

VER="${INPUT_VERSION:?}"
REPO="${INPUT_REPO:-grafana/field-eng-appenv-deployment}"
BASE="${INPUT_BASE:-main}"
DELETE_AFTER="${INPUT_DELETE:-true}"

if ! printf '%s' "$VER" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.+-]+)?$'; then
  echo "::error::Invalid grafana provider version: $VER"
  exit 1
fi

{
  echo "version=$VER"
  echo "field_eng_repo=$REPO"
  echo "base_ref=$BASE"
  echo "delete_after=$DELETE_AFTER"
} >>"$GITHUB_OUTPUT"

echo "Resolved: version=$VER repo=$REPO base=$BASE delete_after=$DELETE_AFTER dev_override_run_id=${GITHUB_RUN_ID:-unknown}"
