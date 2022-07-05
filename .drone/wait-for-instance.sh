#!/usr/bin/env bash

set -euxo pipefail

getStatus() {
  curl -I -L -s -o /dev/null -w "%{http_code}" "${1}"
}

status=$(getStatus "${1}")
i=0
while [ "${status}" != "200" ]; do
  if [ "${i}" -gt "30" ]; then
    echo "instance never became ready"
    exit 1
  fi
  status=$(getStatus "${1}")
  i=$((i+1))
  sleep 2
done
