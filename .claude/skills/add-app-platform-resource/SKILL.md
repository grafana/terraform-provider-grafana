---
name: add-app-platform-resource
description: "Scaffold a new AppPlatform Terraform resource with schema, tests, examples, and provider registration. Use when adding a new K8s-backed Grafana resource."
argument-hint: "[resource-name or GitHub issue URL]"
disable-model-invocation: true
---

**Current AppPlatform resources:**
!`grep -oE 'appplatform\.[a-zA-Z0-9_]+\(\)' pkg/provider/resources.go`

**Git status:**
!`git status --short | head -5`

---

## Critical Rules

- **ALWAYS call tools**, never describe them as text ("I would read..."). Actually read the file.
- **Read reference files before generating any code**: `references/checklist.md`, `references/gotchas.md`, `references/field-type-mapping.md`.
- **Do NOT write production code during the Plan phase** (Phase 0–1). Plan first, get human approval, then implement.
- **Read at least one existing resource file** as a pattern before generating code (see reference table below).
- **TaskCreate/TaskUpdate**: Call these tools at every phase boundary marked **ACTION**. If you catch yourself writing "TaskUpdate..." as text output, STOP and make the actual tool call instead.

---

## Progress Tracking

After exiting plan mode and before starting Phase 2, create a task list using `TaskCreate` for every execute step. Mark each task `in_progress` before starting it and `completed` when done.

**ACTION** — Use `TaskCreate` to create one task per step (adapt based on the approved plan — skip tasks that don't apply):

| # | Subject | activeForm |
|---|---------|------------|
| 1 | Install SDK dependency | Installing SDK dependency |
| 2 | Create resource implementation | Creating resource implementation |
| 3 | Register resource in provider | Registering in AppPlatformResources |
| 4 | Create example HCL | Creating example HCL |
| 5 | Create acceptance test | Creating acceptance test |
| 6 | Update test gating in examples_test.go | Updating test gating |
| 7 | Build and verify compilation | Running go build |
| 8 | Generate documentation | Running make docs |

Skip #1 if the SDK package is already in `go.mod`. Skip #6 if no new version gating is needed.

---

## Phase 0: Gather Requirements

> Enter `/plan` mode before starting Phase 0. Stay in plan mode until the human approves in Phase 1.

### Step 0.1 — Parse the argument

If `$ARGUMENTS` is:
- A GitHub issue URL (`https://github.com/...`) → fetch with `gh issue view <url>`
- A bare integer → fetch with `gh issue view <number>`
- A resource name string → use directly as the resource name

### Step 0.2 — Ask for core identity

Use `AskUserQuestion` to collect (all required):

| Question | Header | Options / Hint |
|----------|--------|----------------|
| API group (e.g. `alerting.grafana.app`) | `API Group` | free text |
| Kind (e.g. `AlertEnrichment`) | `Kind` | free text |
| Version (e.g. `v1beta1`) | `Version` | free text |
| Go SDK import path (e.g. `github.com/grafana/grafana/apps/alerting/alertenrichment/pkg/apis/alertenrichment/v1beta1`) | `SDK Import` | free text |
| Resource category | `Category` | `CategoryAlerting`, `CategoryGrafanaApps`, `CategoryGrafanaEnterprise`, `CategoryGrafanaOSS` |

### Step 0.3 — Auto-detect SDK types

Spawn an **Explore agent** to:
1. Check `go.mod` for the SDK import path — note whether `go get` will be needed.
2. Search the SDK package for: `<Kind>Kind()` function, `<Kind>Spec` struct, `<Kind>List` type.
3. Read the `<Kind>Spec` struct fields — note field names, Go types, required/optional hints (pointers = optional).

If the SDK package is **not** in `go.mod`, ask:
> "The SDK package `<path>` is not in go.mod. Should I run `go get <path>` to add it, or would you prefer to hand-roll local K8s types?"

If SDK types **don't exist** (no `<Kind>Spec` struct), ask:
> "I couldn't find `<Kind>Spec` in the SDK. Should I hand-roll local K8s types (following the `appo11y_config_resource.go` pattern), or wait for upstream types?"

### Step 0.4 — Ask about advanced features

Use `AskUserQuestion` (multi-select OK):

- **Provenance field** — Does this resource support `disable_provenance`? (See `alertenrichment_resource.go` for the pattern.)
- **PlanModifier** — Custom plan modification needed? (See `secret_secure_value_resource.go` for example.)
- **UpdateDecider** — Skip no-op updates? (Useful when API returns computed fields that differ from plan.)
- **UseConfigSpec** — Read spec from Config not Plan? (For write-only fields like secrets.)

### Step 0.5 — Ask about test gating

Use `AskUserQuestion`:

| Question | Header | Options |
|----------|--------|---------|
| Test category | `Test gate` | `OSS`, `Enterprise` |
| Minimum Grafana version | `Min version` | free text, e.g. `>=12.0.0` |

---

## Phase 1: Schema Design & Plan Approval

### Step 1.1 — Read reference files

Read all three reference files:
- `.claude/skills/add-app-platform-resource/references/checklist.md`
- `.claude/skills/add-app-platform-resource/references/gotchas.md`
- `.claude/skills/add-app-platform-resource/references/field-type-mapping.md`

### Step 1.2 — Read a reference resource

Choose an existing resource based on complexity:

| Resource | When to use |
|----------|-------------|
| `internal/resources/appplatform/playlist_resource.go` | Simple: flat spec, list of objects |
| `internal/resources/appplatform/inhibitionrule_resource.go` | Medium: list attributes with validators |
| `internal/resources/appplatform/secret_keeper_resource.go` | Complex: nested blocks |
| `internal/resources/appplatform/secret_secure_value_resource.go` | WriteOnly fields + PlanModifier |
| `internal/resources/appplatform/appo11y_config_resource.go` | Hand-rolled K8s types |

Read the chosen file now.

Also read `internal/resources/appplatform/alertenrichment_resource_acc_test.go` to understand test structure (especially `importStateIDFunc`).

### Step 1.3 — Design the schema

Using the SDK spec fields from Phase 0.3 and the field-type mapping from the reference file, design:

1. **TF resource name**: apply `grafana_apps_{first-segment-of-group}_{lowercase-kind}_{version}`
   - Example: group=`alerting.grafana.app`, kind=`AlertEnrichment`, version=`v1beta1` → `grafana_apps_alerting_alertenrichment_v1beta1`
2. **SpecAttributes / SpecBlocks**: map each SDK field to its TF schema type (see `references/field-type-mapping.md`)
3. **Model structs**: one `<Name>SpecModel` plus nested models as needed
4. **SpecParser**: TF → Go SDK mapping
5. **SpecSaver**: Go SDK → TF mapping (used only during `ImportState`)

### Step 1.4 — Present plan for approval

Present a structured summary:

```
## Implementation Plan

**TF resource name**: grafana_apps_<group>_<kind>_<version>
**Category**: <category>
**Reference resource**: <file used as template>

### Spec Schema
| Field | TF Type | Required/Optional | Description |
|-------|---------|-------------------|-------------|
| ...   | ...     | ...               | ...         |

### Advanced features
- Provenance: yes/no
- PlanModifier: yes/no
- UpdateDecider: yes/no
- UseConfigSpec: yes/no

### Test gating
- testutils.Check<OSS|Enterprise>TestsEnabled(t, "<version>")

### Files to create/modify
1. CREATE internal/resources/appplatform/<name>_resource.go
2. CREATE internal/resources/appplatform/<name>_resource_acc_test.go
3. CREATE examples/resources/grafana_apps_<group>_<kind>_<version>/resource.tf
4. MODIFY pkg/provider/resources.go — add appplatform.<FuncName>() to AppPlatformResources()
5. MODIFY internal/resources/examples_test.go — add case for new resource (if new category or special version)
6. GENERATED docs/resources/apps_<group>_<kind>_<version>.md — via make docs
```

Wait for human approval. If changes requested, iterate on the schema design and re-present.

**Exit `/plan` mode** once the human approves.

---

## Phase 2: Generate All Files

### Step 2.1 — Run `go get` if needed

**ACTION** — Call TaskUpdate: set "Install SDK dependency" task to status="in_progress".

If the SDK package was not in `go.mod`:
```bash
go get <import-path>
```

**ACTION** — Call TaskUpdate: set "Install SDK dependency" task to status="completed". (Skip if task wasn't created.)

### Step 2.2 — Create the resource implementation

**ACTION** — Call TaskUpdate: set "Create resource implementation" task to status="in_progress".

Create `internal/resources/appplatform/<name>_resource.go`.

**Structure** (follow the reference resource exactly):
```go
package appplatform

import (...)

// <Name>SpecModel is a model for the <name> spec.
type <Name>SpecModel struct {
    // fields matching SpecAttributes keys (snake_case), tfsdk tags
}

// <Name>() creates a new Grafana <Name> resource.
func <Name>() NamedResource {
    return NewNamedResource[*<pkg>.<Kind>, *<pkg>.<Kind>List](
        common.<Category>,
        ResourceConfig[*<pkg>.<Kind>]{
            Kind: <pkg>.<Kind>Kind(),
            Schema: ResourceSpecSchema{
                Description:         "...",
                MarkdownDescription: `...`,
                SpecAttributes: map[string]schema.Attribute{
                    // ... fields
                },
                // SpecBlocks if needed
            },
            SpecParser: func(ctx context.Context, src types.Object, dst *<pkg>.<Kind>) diag.Diagnostics {
                var data <Name>SpecModel
                if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
                    UnhandledNullAsEmpty:    true,
                    UnhandledUnknownAsEmpty: true,
                }); diag.HasError() {
                    return diag
                }
                // map data → dst.SetSpec(...)
                return diag.Diagnostics{}
            },
            SpecSaver: func(ctx context.Context, src *<pkg>.<Kind>, dst *ResourceModel) diag.Diagnostics {
                // map src.Spec → types.Object and set dst.Spec
                return diag.Diagnostics{}
            },
        })
}
```

Key rules from `references/gotchas.md`:
- Always use `basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}` in SpecParser
- In SpecSaver, build `types.ObjectValueFrom(ctx, map[string]attr.Type{...}, &data)` — AttrTypes keys MUST match SpecAttributes keys exactly
- Handle nullable optional fields with `types.StringNull()` / `types.ListNull()`, not zero values
- If hand-rolling types, follow `appo11y_config_resource.go` pattern (local Object/ListObject/Kind/Codec types)

**ACTION** — Call TaskUpdate: set "Create resource implementation" task to status="completed".

### Step 2.3 — Register in provider

**ACTION** — Call TaskUpdate: set "Register resource in provider" task to status="in_progress".

Edit `pkg/provider/resources.go`, add to `AppPlatformResources()`:

```go
func AppPlatformResources() []appplatform.NamedResource {
    return []appplatform.NamedResource{
        // ... existing resources ...
        appplatform.<Name>(),  // ADD THIS LINE
    }
}
```

**ACTION** — Call TaskUpdate: set "Register resource in provider" task to status="completed".

### Step 2.4 — Create example

**ACTION** — Call TaskUpdate: set "Create example HCL" task to status="in_progress".

Create `examples/resources/grafana_apps_<group>_<kind>_<version>/resource.tf`:

```hcl
resource "grafana_apps_<group>_<kind>_<version>" "example" {
  metadata {
    uid = "my-<kind>"
  }

  spec {
    # ... required fields with realistic values
    # optional fields can be omitted or included with comments
  }
}
```

**ACTION** — Call TaskUpdate: set "Create example HCL" task to status="completed".

### Step 2.5 — Create acceptance test

**ACTION** — Call TaskUpdate: set "Create acceptance test" task to status="in_progress".

Create `internal/resources/appplatform/<name>_resource_acc_test.go`:

```go
package appplatform_test

import (
    "fmt"
    "testing"

    "github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
    terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
)

const <name>ResourceName = "grafana_apps_<group>_<kind>_<version>.test"

func TestAcc<Name>_basic(t *testing.T) {
    testutils.Check<OSS|Enterprise>TestsEnabled(t, "<version>")

    randSuffix := acctest.RandString(6)

    terraformresource.ParallelTest(t, terraformresource.TestCase{
        ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
        Steps: []terraformresource.TestStep{
            {
                Config: testAcc<Name>Basic(randSuffix),
                Check: terraformresource.ComposeTestCheckFunc(
                    terraformresource.TestCheckResourceAttrSet(<name>ResourceName, "id"),
                    // ... attribute checks
                ),
            },
            {
                ResourceName:      <name>ResourceName,
                ImportState:       true,
                ImportStateVerify: true,
                ImportStateVerifyIgnore: []string{
                    "options.%",
                    "options.overwrite",
                },
                ImportStateIdFunc: importStateIDFunc(<name>ResourceName),
            },
        },
    })
}

func testAcc<Name>Basic(randSuffix string) string {
    return fmt.Sprintf(`
resource "grafana_apps_<group>_<kind>_<version>" "test" {
  metadata {
    uid = "test-%s"
  }
  spec {
    # ... required fields
  }
}
`, randSuffix)
}
```

Note: `importStateIDFunc` is defined in `alertenrichment_resource_acc_test.go` and is available to all tests in the `appplatform_test` package.

**ACTION** — Call TaskUpdate: set "Create acceptance test" task to status="completed".

### Step 2.6 — Update examples_test.go (if needed)

**ACTION** — Call TaskUpdate: set "Update test gating in examples_test.go" task to status="in_progress" (skip if no gating change needed).

If the new resource belongs to a **new category** not already in `examples_test.go`, or needs **special version gating**, add a case:

```go
case strings.Contains(filename, "grafana_apps_<group>_<kind>"):
    testutils.Check<OSS|Enterprise>TestsEnabled(t, "<version>")
```

Find the appropriate category block in `internal/resources/examples_test.go`.

**ACTION** — Call TaskUpdate: set "Update test gating in examples_test.go" task to status="completed" (if applicable).

---

## Phase 3: Build & Verify

### Step 3.1 — Build

**ACTION** — Call TaskUpdate: set "Build and verify compilation" task to status="in_progress".

```bash
go build .
```

If the build fails:
1. Read the error carefully
2. Fix the issue (type mismatch, missing import, wrong AttrTypes keys)
3. Re-run `go build .`
4. Repeat until green

Common build failures (see `references/gotchas.md`):
- `AttrTypes` map keys don't match `SpecAttributes` keys → align them
- Missing import for `basetypes`, `attr`, `diag` → add imports
- Wrong type in model struct (e.g. `types.Int64` instead of `types.Int64Type`) → check field-type-mapping

**ACTION** — Call TaskUpdate: set "Build and verify compilation" task to status="completed".

### Step 3.2 — Generate docs

**ACTION** — Call TaskUpdate: set "Generate documentation" task to status="in_progress".

```bash
make docs
```

Verify `docs/resources/apps_<group>_<kind>_<version>.md` was created.

**ACTION** — Call TaskUpdate: set "Generate documentation" task to status="completed".

---

## Phase 4: Test (Optional)

Ask the human:

> "Would you like to run the acceptance test? This requires a live Grafana instance with `GRAFANA_URL`, `GRAFANA_AUTH`, and `GRAFANA_VERSION` set, plus `TF_ACC_<OSS|ENTERPRISE>=true`."

If yes, run:
```bash
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin TF_ACC=1 TF_ACC_<OSS|ENTERPRISE>=true GRAFANA_VERSION=<version> \
  go test ./internal/resources/appplatform/... -run TestAcc<Name> -v -timeout 30m
```

---

## Phase 5: Summary

**ACTION** — Call `TaskList` and verify all tasks are marked completed. If any were skipped or failed, note them in the summary.

Present a summary:

```
## Summary

### Files created
- internal/resources/appplatform/<name>_resource.go
- internal/resources/appplatform/<name>_resource_acc_test.go
- examples/resources/grafana_apps_<group>_<kind>_<version>/resource.tf

### Files modified
- pkg/provider/resources.go (added appplatform.<Name>() to AppPlatformResources)
- internal/resources/examples_test.go (if gating was added)

### Generated
- docs/resources/apps_<group>_<kind>_<version>.md (via make docs)

### Next steps
- Review the generated schema and adjust descriptions as needed
- Commit and open a PR
- Consider adding more test cases for edge cases / update scenarios
```

---

## Reference Files for This Skill

| File | Purpose |
|------|---------|
| `internal/resources/appplatform/resource.go` | Generic framework — ResourceConfig, ResourceModel, NamedResource |
| `internal/resources/appplatform/playlist_resource.go` | Simple resource (flat spec + list of objects) |
| `internal/resources/appplatform/inhibitionrule_resource.go` | Medium complexity (list attributes with validators) |
| `internal/resources/appplatform/secret_keeper_resource.go` | Nested blocks example |
| `internal/resources/appplatform/secret_secure_value_resource.go` | WriteOnly fields + PlanModifier |
| `internal/resources/appplatform/appo11y_config_resource.go` | Hand-rolled K8s types (no SDK) |
| `internal/resources/appplatform/alertenrichment_resource_acc_test.go` | Test patterns + `importStateIDFunc` |
| `pkg/provider/resources.go` | Registration point (AppPlatformResources at line ~83) |
| `internal/resources/examples_test.go` | Example test gating |
| `internal/common/resource.go` | Category constants |
