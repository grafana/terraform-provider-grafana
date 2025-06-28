# Issue Template Generator

A simple script to generate an enhanced bug report template (based on the original) with a dropdown list of terraform resources.

## Overview

This generates a one-time enhanced version of the existing bug report template (`3-bug-report-enhanced.yml`) that replaces the manual textarea with a proper dropdown for selecting terraform resources and data sources.

## How It Works

The script sources terraform resources in priority order:

1. **Catalog Files** (when PR #2228 merges) - Reads `internal/resources/*/catalog-*.yaml` files (tested by pulling down #2228)
2. **Hardcoded List** (current fallback) - Uses exact names from PR #2228
3. **Future: EnghHub API** - Could query live Backstage API when service account available

### EnghHub/Backstage API Integration (Future Enhancement)

To integrate with Backstage API directly, we'll need a Service Account and to set these environment variables:
```bash
export BACKSTAGE_TOKEN and BACKSTAGE_URL="https://enghub.grafana-ops.net"  # optional
go run main.go
```

The tool would then query:
- `{BACKSTAGE_URL}/api/catalog/entities?filter=kind=component,spec.type=terraform-resource`
- `{BACKSTAGE_URL}/api/catalog/entities?filter=kind=component,spec.type=terraform-data-source`
