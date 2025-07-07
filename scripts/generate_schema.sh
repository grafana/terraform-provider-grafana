#!/usr/bin/env bash

CURRENTDIR=$(realpath $(dirname $0))
REPODIR="${CURRENTDIR}"/..
WORKDIR=$(mktemp -d)
function finish {
    rm -rf "${WORKDIR}"
}
trap finish EXIT

cd "$REPODIR"
go build .


cd "${WORKDIR}"

echo '
provider_installation {
   dev_overrides {
      "grafana/grafana" = "'${REPODIR}'" # this path is the directory where the binary is built
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}' > terraform.rc


echo '{"terraform":[{"required_providers":[{"provider":{"source":"grafana/grafana"}}]}]}' > main.tf.json

export TF_CLI_CONFIG_FILE=terraform.rc

terraform providers schema -json=true
