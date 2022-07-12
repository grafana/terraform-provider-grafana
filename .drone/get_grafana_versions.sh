#!/bin/bash

gh api 'repos/grafana/grafana/releases' \
  --jq '.[].tag_name[1:6]' \
    | sort -r \
    | uniq \
    | jq \
      --slurp \
      --raw-input 'split("\n")[:-1]'
