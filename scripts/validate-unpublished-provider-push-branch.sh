#!/usr/bin/env bash
# Clones field-eng-appenv-deployment at BASE_REF and pushes an ephemeral branch at the same commit.
#
# Environment:
#   GH_TOKEN — GATB app token (contents write on field-eng)
#   FIELD_ENG_REPO, BASE_REF — required
#   GITHUB_RUN_ID, GITHUB_SHA — from this workflow run; used only for unique branch naming
#   GITHUB_OUTPUT — writes branch=<name>

set -euo pipefail

: "${GH_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${BASE_REF:?}"

RUN_ID="${GITHUB_RUN_ID:?}"
SHORT_SHA="${GITHUB_SHA:-manual}"
SHORT_SHA="${SHORT_SHA:0:7}"
BRANCH="tpg-validate-${RUN_ID}-${SHORT_SHA}"
BRANCH="$(printf '%s' "$BRANCH" | tr -cd 'a-zA-Z0-9._-')"

git clone --depth 1 --branch "$BASE_REF" "https://x-access-token:${GH_TOKEN}@github.com/${FIELD_ENG_REPO}.git" fe-appenv
cd fe-appenv

git push "https://x-access-token:${GH_TOKEN}@github.com/${FIELD_ENG_REPO}.git" "HEAD:refs/heads/${BRANCH}"

echo "branch=${BRANCH}" >>"$GITHUB_OUTPUT"
echo "Pushed ${FIELD_ENG_REPO}@${BRANCH} (same commit as ${BASE_REF})"
