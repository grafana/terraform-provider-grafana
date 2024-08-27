#!/bin/bash

PATCH_FILE="0001-testing-commit.patch"

if git apply --check ${PATCH_FILE}; then
	echo "applying patch for running against local k3d..."
	git am < ${PATCH_FILE}
else
	echo "there's an error when trying to add the patch"
	echo "Run: git apply --check ${PATCH_FILE}" for details"
	exit 1
fi
