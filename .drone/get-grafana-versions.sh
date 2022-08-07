#!/usr/bin/env bash

# This script is intended to output a JSON array of the
# last 5 minor versions of Grafana for use in dynamically
# templating the .drone/drone.yml with the Grafana versions
# targeted by terraform-provider-grafana tests.
#
# Example:
#
# [
#  "9.1.0",
#  "9.0.6",
#  "8.5.9",
#  "8.4.7",
#  "8.3.7"
# ]
#
# The command seeks to do the following:
#
# 1. fetch Grafana releases
# 2. create a selection of their tag names, stripping out
#     the 'v' prefix and possible '-X'-style prerelease
#     build metadata suffixes.
# 3. remove duplicates resulting from multiple major.minor.patch
#     versions with unique '-X'-style prerelease build metadata
# 4. split on line breaks and turn into an array
# 5. map each element to a {"major": "x", "minor": "y", "patch": "z"}
#     JSON object.
# 6. subgroup each release into an array of its minor releases
# 7. reverse the array to be in descending order
# 8. map the result to an array of the latest 5 minor Grafana versions

gh api 'repos/grafana/grafana/releases' \
  --jq '.[].tag_name[1:6]' \
    | sort -r \
    | uniq \
    | jq \
      --slurp \
      --raw-input \
      'split("\n")[:-1]
      | map({
        major: (split(".")[0]),
        minor: (split(".")[1]),
        patch: (split(".")[2])
      })
      | group_by(.major, .minor)
      | reverse
      | map(.[-1] | join("."))[:5]
    '
