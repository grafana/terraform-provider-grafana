#!/bin/sh

set -e

version=$1
baseUrl="https://github.com/grafana/grafana-app-sdk/releases/download/v${version}"

if [[ -z $version ]]; then
  echo "Please specify which version you want to install!"
  exit 1

  # TODO: once the SDK releases remove the version from the asset name,
  # it should be possible to install the latest one.
  #
  # version="latest"
  # baseUrl="https://github.com/grafana/grafana-app-sdk/releases/latest/download"
fi

outDir=$2
if [[ -z $outDir  ]]; then
  outDir="./bin"
fi

platform=""
case $(uname -s) in
  Linux*) platform=linux;;
  Darwin*) platform=darwin;;
esac

if [[ -z $platform  ]]; then
  echo "Unsupported platform!"
  exit 1
fi

arch=""
case $(uname -m) in
  x86_64) arch="amd64" ;;
  arm64) arch="arm64" ;;
  arm) arch="arm64" ;;
esac

if [[ -z $arch  ]]; then
  echo "Unsupported architecure!"
  exit 1
fi

asset="grafana-app-sdk_${version}_${platform}_${arch}.tar.gz"
url="${baseUrl}/${asset}"

echo "Will download from: ${url} into ${outDir}"
curl -Ls "${url}" | tar xvzf - -C "${outDir}"
rm -f "${outDir}/LICENSE" "${outDir}/README.md"
mv "${outDir}/grafana-app-sdk" "${outDir}/grafana-app-sdk-${version}"

# TODO: compare checksums
