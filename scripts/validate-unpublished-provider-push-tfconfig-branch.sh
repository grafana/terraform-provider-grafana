#!/usr/bin/env bash
# Clones field-eng-appenv-deployment, bumps grafana/grafana version in tfconfig.jsonnet, pushes ephemeral branch.
#
# Environment:
#   GH_TOKEN, FIELD_ENG_REPO, BASE_REF, TPG_VERSION — required
#   GITHUB_RUN_ID, GITHUB_SHA — for branch naming
#   GITHUB_OUTPUT — writes branch=<name>

set -euo pipefail

: "${GH_TOKEN:?}"
: "${FIELD_ENG_REPO:?}"
: "${BASE_REF:?}"
: "${TPG_VERSION:?}"

RUN_ID="${GITHUB_RUN_ID:?}"
SHORT_SHA="${GITHUB_SHA:-manual}"
SHORT_SHA="${SHORT_SHA:0:7}"
BRANCH="tpg-validate-${TPG_VERSION}-${RUN_ID}-${SHORT_SHA}"
BRANCH="$(printf '%s' "$BRANCH" | tr -cd 'a-zA-Z0-9._-')"

git clone --depth 1 --branch "$BASE_REF" "https://x-access-token:${GH_TOKEN}@github.com/${FIELD_ENG_REPO}.git" fe-appenv
cd fe-appenv

export TPG_VERSION
perl -i -0pe 's/("configuration_aliases": \["grafana\.stack", "grafana\.cloud"\],\s*\n\s*"version": ")[^"]+/${1}$ENV{TPG_VERSION}/' tfconfig.jsonnet

if git diff --quiet; then
  echo "::error::Expected tfconfig.jsonnet to change; check the perl regex still matches field-eng-appenv-deployment."
  exit 1
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add tfconfig.jsonnet
git commit -m "chore: set grafana provider to ${TPG_VERSION} (TPG CI validation)"
git push "https://x-access-token:${GH_TOKEN}@github.com/${FIELD_ENG_REPO}.git" "HEAD:refs/heads/${BRANCH}"

echo "branch=${BRANCH}" >>"$GITHUB_OUTPUT"
echo "Pushed ${FIELD_ENG_REPO}@${BRANCH}"
