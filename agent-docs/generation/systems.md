# Code Generation Systems

Two independent code generation systems serve different purposes.

## System 1: Documentation Generation (`go generate ./...`)

Triggered by `main.go` go:generate directives. Run after any schema/example change.

```sh
make docs   # or: go generate ./...
```

Three sequential steps:

### Step 1: genimports
```
go run ./tools/genimports examples
```
Generates `examples/resources/<name>/import.sh` for each resource where `IDType != nil`:

```bash
# Example output for grafana_folder:
terraform import grafana_folder.name "{{ uid }}"
terraform import grafana_folder.name "{{ orgID }}:{{ uid }}"
```

The second command is generated when optional fields exist. Source: `r.ImportExample()` in `internal/common/resource.go:117`.

### Step 2: tfplugindocs
```
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
    --provider-name "terraform-provider-grafana"
```

Compiles the provider, extracts all schema `Description` fields, merges with:
- `templates/` — Markdown templates with `{{.SchemaMarkdown}}`, `{{ tffile "..." }}`, `{{ codefile "shell" "..." }}`
- `examples/resources/<name>/resource.tf` — embedded in docs
- `examples/resources/<name>/import.sh` — from step 1

Writes `docs/resources/*.md` and `docs/data-sources/*.md`. **Never edit `docs/` manually.** CI fails if `go generate ./...` produces any git diff.

### Step 3: setcategories
```
go run ./tools/setcategories docs
```
Patches the `subcategory` frontmatter in each generated Markdown file from the resource's `ResourceCategory`:

```yaml
# Before:  subcategory: ""
# After:   subcategory: "Grafana OSS"
```

`tfplugindocs` always emits empty subcategory. This step fills it from the `ResourceCommon.Category` field.

## System 2: Config Generation (`pkg/generate/`)

A runtime tool (`cmd/generate/`) that connects to a live Grafana instance and generates Terraform HCL representing all existing resources. Separate from documentation generation.

```sh
terraform-provider-grafana-generate [flags]
```

### End-to-End Pipeline

```
Generate(ctx, config)
│
├── Write provider.tf (required_providers block)
├── Install Terraform binary (default v1.8.5 via hc-install)
├── terraform init
│
├── generateCloudResources(ctx, client, cfg)   ← cloud resources first
│     └── generateImportBlocks(...)
└── generateGrafanaResources(ctx, client, cfg) ← then Grafana resources
      └── generateImportBlocks(...)
            │
            ├── [parallel] For each resource with ListIDsFunc:
            │     call ListIDsFunc(ctx, client, listerData) → []string IDs
            │
            ├── Write imports.tf (import { to = ...; id = "..." } blocks)
            │
            ├── terraform plan -generate-config-out=resources.tf
            │     (Terraform calls provider.Read for each resource, writes HCL)
            │
            └── Post-processing pipeline (8 passes, in order):
                  1. ReplaceNullSensitiveAttributes  → "SENSITIVE_VALUE_TO_REPLACE"
                  2. removeOrphanedImports           → remove import blocks with no resource
                  3. UsePreferredResourceNames       → "grafana_stack.12345" → "grafana_stack.my_name"
                  4. sortResourcesFile               → alphabetical sort
                  5. WrapJSONFieldsInFunction        → JSON strings → jsonencode({...})
                  6. StripDefaults                   → remove null/empty/default attributes
                  7. ExtractDashboards               → move large JSON to separate files
                  8. ReplaceReferences               → literal IDs → Terraform cross-references
```

### Lister Functions

Every resource supporting generation must have a `ListIDsFunc`:

```go
type ResourceListIDsFunc func(ctx context.Context, client *Client, data any) ([]string, error)

// Attach to resource:
common.NewLegacySDKResource(...).WithLister(myListFunc)
```

**Two helper wrappers** in `internal/resources/grafana/common_lister.go`:

```go
// Single-org resource:
listerFunction(func(ctx, client *goapi.GrafanaHTTPAPI) ([]string, error) { ... })

// Org-scoped resource (iterates all orgs):
listerFunctionOrgResource(func(ctx, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) { ... })
```

### ListerData Pattern

A shared `data any` object is passed to all lister calls. Uses `sync.Once` for lazy caching:

```go
// grafana.ListerData — caches list of all org IDs (fetched once for all org-scoped listers)
// cloud.ListerData  — caches list of all stacks and cloud org ID
```

All parallel lister goroutines share the same `listerData` instance, so expensive API calls (e.g., "list all orgs") are made exactly once per generation run.

### Crossplane Mode

When `Format = "crossplane"`:
1. Normal TF generation runs first
2. `convertToTFJSON()` converts `.tf` → `.tf.json`
3. Each resource is re-emitted as a Crossplane YAML manifest
4. Resource `Category` maps to Crossplane `apiVersion` (e.g., `CategoryCloud` → `cloud.grafana.crossplane.io/v1alpha1`)
5. All Terraform files are deleted — only YAML remains

### Reference Table Auto-Generation

The static cross-reference table in `pkg/generate/replace_references.go` (145+ entries like `"grafana_dashboard.folder=grafana_folder.id"`) is regenerated by:

```sh
go generate  # inside pkg/generate/replace_references.go
```

`genreferences/main.go` walks all `.tf` files and test files, regex-extracts cross-resource attribute assignments, and rewrites the `var knownReferences` slice.

### cmd/without-lister

A dev utility in `cmd/without-lister/` lists all registered resources that are missing a `ListIDsFunc`. Run this to find resources that don't support config generation yet.
