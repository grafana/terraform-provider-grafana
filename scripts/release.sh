#!/usr/bin/env bash
set -euo pipefail

# Verify we're on the main branch.
branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$branch" != "main" ]; then
	echo "Error: releases must be created from the main branch (currently on '$branch')."
	exit 1
fi

# Fetch latest state from origin.
git fetch origin main --tags --quiet

# Prompt to pull if local is behind.
local_sha=$(git rev-parse HEAD)
remote_sha=$(git rev-parse origin/main)
if [ "$local_sha" != "$remote_sha" ]; then
	echo "Local branch is behind origin/main."
	printf "Pull latest changes? [y/N] "
	read -r ans
	if [ "$ans" = "y" ] || [ "$ans" = "Y" ]; then
		git pull --ff-only origin main
	else
		echo "Continuing with local HEAD."
	fi
fi

# Determine the release version.
if [ -n "${RELEASE_VERSION:-}" ]; then
	echo "Using provided version: $RELEASE_VERSION"
elif command -v git-cliff >/dev/null 2>&1; then
	RELEASE_VERSION=$(git cliff --bumped-version 2>/dev/null)
	latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || true)
	if [ "$RELEASE_VERSION" = "$latest_tag" ]; then
		echo "Error: no unreleased changes to bump. Nothing to release."
		exit 1
	fi
	echo "Computed version from conventional commits: $RELEASE_VERSION"
else
	echo "Error: git-cliff is not installed and RELEASE_VERSION is not set."
	echo "Install git-cliff: https://git-cliff.org/docs/installation"
	exit 1
fi

# Create and push the tag.
git tag "$RELEASE_VERSION"
git push origin "$RELEASE_VERSION"
