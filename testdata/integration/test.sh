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

rm -rf ${SCRIPT_DIR}/generated ${SCRIPT_DIR}/generated-json ${SCRIPT_DIR}/config.tfrc ${SCRIPT_DIR}/terraform.tfstate*

# Build the provider
cd ${SCRIPT_DIR}/../..
REPO_ROOT=$(pwd)
go build

# Build the code generator tool
cd "${REPO_ROOT}/cmd/generate"
go build -o "${REPO_ROOT}/terraform-provider-grafana-generate"

# Write the Terraform configuration (points to the locally built provider)
cd ${SCRIPT_DIR}
rm -rf .terraform* terraform.state && \
cat <<EOF > config.tfrc
provider_installation {
  dev_overrides {
     "grafana/grafana" = "${REPO_ROOT}"
  }
  direct {}
}
EOF

# Run Terraform
export TF_CLI_CONFIG_FILE=${SCRIPT_DIR}/config.tfrc
export GRAFANA_URL=http://0.0.0.0:3000
export GRAFANA_VERSION=${GRAFANA_VERSION}

trap "docker compose down" EXIT
docker compose up ${DOCKER_COMPOSE_ARGS}
terraform apply -auto-approve

# Test code generation
${REPO_ROOT}/terraform-provider-grafana-generate \
  --terraform-provider-version "v3.0.0" \
  --grafana-url ${GRAFANA_URL} \
  --grafana-auth "admin:admin" \
  --clobber \
  --output-dir ${SCRIPT_DIR}/generated \
  --include-resources "grafana_folder.*" \
  --include-resources "grafana_team.*" \
  --output-credentials

${REPO_ROOT}/terraform-provider-grafana-generate \
  --terraform-provider-version "v3.0.0" \
  --grafana-url ${GRAFANA_URL} \
  --grafana-auth "admin:admin" \
  --clobber \
  --output-dir ${SCRIPT_DIR}/generated-json \
  --output-format json \
  --include-resources "grafana_folder.*" \
  --include-resources "grafana_team.*" \
  --output-credentials

# Test the generated code
for dir in "generated" "generated-json" ; do
  cd ${SCRIPT_DIR}/${dir}
  terraform plan | tee plan.out
  # Expect a folder called "My Folder" and no changes in the plan
  grep "My Folder" plan.out || (echo "Expected a folder called 'My Folder'" && exit 1)
  grep ' to import, 0 to add, 0 to change, 0 to destroy' plan.out || (echo "Expected no changes in the plan" && exit 1)
done

cd ${SCRIPT_DIR}
terraform destroy -auto-approve
