# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Commands

```sh
# Build
go build .

# Run unit tests (no Grafana instance required)
go test ./... -run TestUnit

# Run all acceptance tests against a live Grafana instance
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin make testacc

# Run OSS acceptance tests (most common during development)
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin make testacc-oss

# Run OSS acceptance tests with Docker Compose (starts Grafana automatically)
make testacc-oss-docker

# Run Enterprise acceptance tests with Docker Compose
make testacc-enterprise-docker

# Lint (runs in Docker)
make golangci-lint

# Regenerate docs (must be run after schema changes)
make docs   # or: go generate ./...
```

### Running a single acceptance test

```sh
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin TF_ACC=1 TF_ACC_OSS=true GRAFANA_VERSION=11.0.0 \
  go test ./internal/resources/grafana/... -run TestAccDashboard_basic -v -timeout 30m
```

The test binary must be built first — the `testacc` Makefile target handles this automatically.

## Architecture

The provider is a **muxed provider** that combines two Terraform plugin frameworks:

```
main.go
└── pkg/provider/provider.go: MakeProviderServer()
    ├── Legacy SDKv2 provider  (pkg/provider/legacy_provider.go)  ← most existing resources
    └── Plugin Framework provider (pkg/provider/framework_provider.go) ← newer resources
```

Both share the same `*common.Client` passed as provider meta.

### Directory structure

```
pkg/provider/         Provider wiring: mux server, client creation, resource registration
  configure_clients.go  Constructs all API clients from provider config
  resources.go          Aggregates all Resources/DataSources from internal packages

internal/common/      Shared types used across all resource packages
  client.go             common.Client struct — holds all API clients (GrafanaAPI, CloudAPI, SMAPI, etc.)
  resource.go           Resource/DataSource wrappers + NewLegacySDKResource / NewResource constructors
  resource_id.go        ResourceID type for typed composite IDs (e.g., "orgID:uid")
  schema.go             Schema helpers (CloneResourceSchemaForDatasource, validators, etc.)

internal/resources/   One package per Grafana product domain:
  grafana/              Core Grafana OSS + Enterprise (dashboards, folders, alerting, users, etc.)
  cloud/                Grafana Cloud API (stacks, access policies, etc.)
  oncall/               Grafana OnCall
  machinelearning/      Machine Learning
  slo/                  SLO
  syntheticmonitoring/  Synthetic Monitoring
  cloudprovider/        Cloud Provider (AWS integration)
  connections/          Connections API
  fleetmanagement/      Fleet Management
  frontendo11y/         Frontend Observability
  asserts/              Asserts / Knowledge Graph
  k6/                   k6 load testing
  appplatform/          App Platform (Kubernetes-backed resources)

internal/testutils/   Test helpers
  provider.go           ProtoV5ProviderFactories, CheckOSSTestsEnabled, CheckEnterpriseTestsEnabled, etc.
  lister.go             CheckLister() — validates resource lister functions in tests

examples/             Terraform HCL examples used both as documentation source and in acceptance tests
  resources/<name>/   _acc_*.tf files are loaded by TestAccExample() in tests
  data-sources/<name>/

docs/                 Generated docs — do NOT edit manually; use `make docs`
templates/            tfplugindocs templates that drive doc generation
```

### Resource anatomy

Every resource/datasource is registered as a `*common.Resource` or `*common.DataSource`:

```go
// SDKv2 style (most existing resources)
common.NewLegacySDKResource(common.CategoryGrafanaOSS, "grafana_dashboard", resourceID, schemaResource).
    WithLister(listDashboards)

// Plugin Framework style (newer resources)
common.NewResource(common.CategoryGrafanaOSS, "grafana_folder", resourceID, &folderResource{})
```

Resources within a package are collected into package-level `Resources` and `DataSources` slices and aggregated in `pkg/provider/resources.go`.

### Org-scoped resource IDs

Most core Grafana resources use composite IDs of the form `<orgID>:<resourceID>` (e.g., `"1:basic"` for a dashboard). Helper functions in `internal/resources/grafana/oss_org_id.go`:

- `OAPIClientFromExistingOrgResource(meta, id)` — used in Read/Update/Delete
- `OAPIClientFromNewOrgResource(meta, d)` — used in Create (reads `org_id` attribute)
- `MakeOrgResourceID(orgID, resourceID)` / `SplitOrgResourceID(id)` — construct/parse IDs

### Acceptance test gating

Tests are skipped unless the appropriate env var is set:

| Env var | Test category |
|---|---|
| `TF_ACC_OSS=true` | OSS features (also needs `GRAFANA_URL`, `GRAFANA_AUTH`, `GRAFANA_VERSION`) |
| `TF_ACC_ENTERPRISE=true` | Enterprise features |
| `TF_ACC_CLOUD_API=true` | Cloud API (needs `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN`, `GRAFANA_CLOUD_ORG`) |
| `TF_ACC_CLOUD_INSTANCE=true` | Cloud instance features (SM, OnCall, k6, etc.) |

Always call the appropriate `testutils.CheckOSSTestsEnabled(t)` / `CheckEnterpriseTestsEnabled(t)` / etc. as the **first line** of every acceptance test.

### Documentation

- Schema `Description` fields + `examples/` + `templates/` → `docs/` via `go generate ./...`
- Example files prefixed with `_acc_` are also used as acceptance test configs via `testutils.TestAccExample(t, "resources/grafana_foo/_acc_basic.tf")`
- After any schema or example change, run `make docs` and commit the updated `docs/` files

### Concurrency

Some resources use mutexes on the shared `common.Client` to avoid race conditions during parallel applies. Wrap CRUD functions with the appropriate helper:

```go
common.WithAlertingMutex[schema.CreateContextFunc](createFn)
common.WithDashboardMutex[schema.UpdateContextFunc](updateFn)
common.WithFolderMutex[schema.DeleteContextFunc](deleteFn)
```

## Codebase Architecture Insights

*Generated by comprehensive analysis (10 domains, ~450 files, 95% confidence)*

### Three-Layer Resource Architecture

This is the most critical concept. Three distinct patterns coexist — always identify which layer you're working in before making changes:

| Layer | Count | When to Use | Registration |
|-------|-------|-------------|-------------|
| **SDKv2** (legacy) | ~65 resources | Existing OSS/Enterprise/Cloud/OnCall/ML/SLO/SM resources | `NewLegacySDKResource(...)` sets `.Schema` |
| **Plugin Framework** | ~26 resources | All new REST-based resources; k6, fleet, connections, cloudprovider, frontendo11y | `NewResource(...)` sets `.PluginFrameworkSchema` |
| **AppPlatform generic** | 11 resources | Grafana App Platform K8s-backed resources (dashboard v2, alerting, secrets) | `AppPlatformResources()` — bypasses `*common.Resource` entirely |

**Decision rule:** New resource? Use Plugin Framework. App Platform K8s API? Use AppPlatform generic. Modifying existing SDKv2 resource? Stay in SDKv2.

Detailed docs: `agent-docs/resources/`

### Mux Provider Assembly

```
main.go
  └── MakeProviderServer()                          [pkg/provider/provider.go]
        ├── FrameworkProvider(v6) → tf6to5server → proto v5 adapter
        ├── Provider(SDKv2, native v5)
        └── tf5muxserver routes by resource type name
```

Both sub-providers call `CreateClients()` independently to produce equivalent (but separate) `*common.Client` instances.

### API Client Architecture

`*common.Client` (`internal/common/client.go`) holds 13+ API clients, each conditionally created:

- **OpenAPI REST** (`/api/*`): `GrafanaAPI`, `GrafanaCloudAPI`, `SLOClient`, `K6APIClient`, `AssertsAPIClient`
- **K8s App Platform** (`/apis/*`): `GrafanaAppPlatformAPI` — completely different transport stack
- **Hand-written REST**: `CloudProviderAPI`, `ConnectionsAPIClient`, `FrontendO11yAPIClient`
- **ConnectRPC/gRPC**: `FleetManagementClient`
- **Others**: `SMAPI`, `MLAPI`, `OnCallClient`

Auth modes: `user:password` (basic), single token (bearer), `"anonymous"`. Each service can have its own token.

Detailed docs: `agent-docs/provider/api-clients.md`

### AppPlatform Generic Resource Pattern

Resources in `internal/resources/appplatform/` use `Resource[T sdkresource.Object, L sdkresource.ListObject]`:

```go
type ResourceConfig[T sdkresource.Object] struct {
    Schema        ResourceSpecSchema    // TF schema
    Kind          sdkresource.Kind     // API group/version/kind
    SpecParser    SpecParser[T]        // TF state → K8s object (create/update)
    SpecSaver     SpecSaver[T]         // K8s object → TF state (import only)
    PlanModifier  ResourcePlanModifier // optional
    UpdateDecider ResourceUpdateDecider // optional: skip no-op updates
    UseConfigSpec bool                 // read spec from Config not Plan (write-only fields)
}
```

Key behaviors:
- **Read does NOT refresh spec** from API — preserves state to avoid false diffs
- **SpecSaver only called during ImportState**
- ID is K8s UUID (`metadata.uuid`); import by `metadata.uid` (K8s name)
- Namespace: `stacks-<stackID>` (cloud) or `org-<orgID>` (local)
- Resource names: `grafana_apps_<group>_<kind>_<version>`

Detailed docs: `agent-docs/resources/appplatform.md`

### Error Handling by Layer

```
SDKv2:        common.CheckReadError("type", d, err)  → calls d.SetId("") on 404
Framework:    resp.Diagnostics.AddError(...)         → resp.State.RemoveResource(ctx) on 404
AppPlatform:  ErrorToDiagnostics(err, resp)          → K8s APIStatus → attribute-level diagnostics
```

### Code Generation

Two independent systems:
1. **Doc generation** (`go generate ./...`): `genimports` → `tfplugindocs` → `setcategories` → `docs/`
2. **Config generation** (`pkg/generate/`): List resources via listers → write import blocks → `terraform plan -generate-config-out` → post-process HCL

Both rely on `ResourceListIDsFunc` lister functions attached via `.WithLister(fn)`.

### Known Gotchas

- `asserts` package uses SDKv2 despite being a newer addition (anomaly — routes via plugin proxy)
- AppPlatform `KeeperActivation` breaks the generic pattern (no spec block, hardcoded name)
- `CloneResourceSchemaForDatasource` shares nested `Elem` references — mutating the clone affects the original
- `util.Ptr[T]` and `common.Ref[T]` are identical helpers (duplication)
