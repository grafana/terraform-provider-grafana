#!/usr/bin/env bash

set -euxo pipefail

# DOCKER_COMPOSE_ARGS and GRAFANA_VERSION need to be set
if [ -z "${DOCKER_COMPOSE_ARGS}" ]; then
  echo "DOCKER_COMPOSE_ARGS is not set"
  exit 1
fi

if [ -z "${GRAFANA_VERSION}" ]; then
  echo "GRAFANA_VERSION is not set"
  exit 1
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Build the provider
cd ${SCRIPT_DIR}/../..
REPO_ROOT=$(pwd)
go build

# Write the Terraform configuration (points to the locally built provider)
cd ${SCRIPT_DIR}
rm -rf .terraform* terraform.state && \
cat <<EOF > config.tfrc
provider_installation {
   dev_overrides {
     "grafana/grafana" = "${REPO_ROOT}"
  }
}
EOF

# Run Terraform
export TF_CLI_CONFIG_FILE=${SCRIPT_DIR}/config.tfrc
export GRAFANA_URL=http://0.0.0.0:3000
export GRAFANA_VERSION=${GRAFANA_VERSION}

trap "docker compose down" EXIT
docker compose up ${DOCKER_COMPOSE_ARGS}
terraform apply -auto-approve
terraform destroy -auto-approve
