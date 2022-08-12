#!/usr/bin/env bash

# This script is intended to output a JSON array of the
# latest patch release of each of the latest 5 minor
# versions of Grafana. It's intended for use in dynamically
# templating the .drone/drone.yml with the Grafana versions
# targeted by terraform-provider-grafana tests.
#
# Example output:
#
# [
#  "9.0.7",
#  "8.5.10",
#  "8.4.10",
#  "8.3.7",
#  "8.2.7"
# ]

gh api 'repos/grafana/grafana/releases?per_page=100' \
  --jq '
    [
      .[]
      | select(.prerelease or .draft | not)
      | .tag_name[1:100]
      | split("-")[0]
    ]
    | map({
      major: (split(".")[0]),
      minor: (split(".")[1]),
      patch: (split(".")[2])
    })
    | group_by(.major, .minor)
    | reverse
    | map(.[0] | join("."))[:5]
    '
