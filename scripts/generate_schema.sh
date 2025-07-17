#!/usr/bin/env bash

CURRENTDIR=$(realpath $(dirname $0))
REPODIR="${CURRENTDIR}"/..
WORKDIR=$(mktemp -d)

TERRAFORM_VERSION="1.12.2"
INSTALL_DIR="${REPODIR}"/testdata/terraform_"${TERRAFORM_VERSION}"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)   OS="linux" ;;
  darwin*)  OS="darwin" ;;
  msys*|cygwin*|mingw*) OS="windows" ;;
  *) echo "Unknown OS: $OS"; exit 1 ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  i386|i686) ARCH="386" ;;
  armv6l|armv7l) ARCH="arm" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unknown Arch: $ARCH"; exit 1 ;;
esac

FILENAME=terraform_"${TERRAFORM_VERSION}"_"${OS}"_"${ARCH}".zip

function finish {
    rm -rf "${WORKDIR}"
}
trap finish EXIT

cd "${REPODIR}"
go build .

cd "${WORKDIR}"

echo '
provider_installation {
   dev_overrides {
      "grafana/grafana" = "'"${REPODIR}"'" # this path is the directory where the binary is built
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}' > terraform.rc


echo '{"terraform":[{"required_providers":[{"provider":{"source":"grafana/grafana"}}]}]}' > main.tf.json

export TF_CLI_CONFIG_FILE=terraform.rc

if [ ! -d "${INSTALL_DIR}" ]; then
  mkdir -p "${INSTALL_DIR}"
  
  curl -o "${FILENAME}" https://releases.hashicorp.com/terraform/"${TERRAFORM_VERSION}"/"${FILENAME}"
  unzip -qq -o "${FILENAME}" -d "${INSTALL_DIR}"
fi

"${INSTALL_DIR}/terraform" providers schema -json=true
