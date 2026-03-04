# Testing Infrastructure

## Test Categories & Gating

Every acceptance test must call a gating helper as its **first line**. Tests are skipped unless the relevant env var is set.

| Helper | Env Var | Additional Required Vars |
|--------|---------|--------------------------|
| `CheckOSSTestsEnabled(t)` | `TF_ACC_OSS=true` | `GRAFANA_URL`, `GRAFANA_AUTH`, `GRAFANA_VERSION` |
| `CheckEnterpriseTestsEnabled(t)` | `TF_ACC_ENTERPRISE=true` | `GRAFANA_URL`, `GRAFANA_AUTH` |
| `CheckCloudAPITestsEnabled(t)` | `TF_ACC_CLOUD_API=true` | `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN`, `GRAFANA_CLOUD_ORG` |
| `CheckCloudInstanceTestsEnabled(t)` | `TF_ACC_CLOUD_INSTANCE=true` | 11+ vars (SM, OnCall, k6, Cloud Provider, Fleet) |
| `IsUnitTest(t)` | inverted: skips when `TF_ACC=true` | none |

All acceptance tests also require the outer `TF_ACC=1` env var (set automatically by Make targets).

Optional semver constraint: `CheckOSSTestsEnabled(t, ">=11.3.0")` skips if `GRAFANA_VERSION` doesn't satisfy the constraint.

## Running Tests

```sh
# All OSS acceptance tests (with Docker — starts Grafana automatically):
make testacc-oss-docker

# All OSS acceptance tests (with your own Grafana instance):
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin make testacc-oss

# Single test:
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin TF_ACC=1 TF_ACC_OSS=true \
  GRAFANA_VERSION=11.0.0 \
  go test ./internal/resources/grafana/... -run TestAccDashboard_basic -v -timeout 30m

# Enterprise tests (Docker):
make testacc-enterprise-docker

# Unit tests only (no Grafana needed):
go test ./... -run TestUnit
```

## ProtoV5ProviderFactories

Defined in `internal/testutils/provider.go:25`. Creates the full muxed provider for acceptance tests:

```go
ProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
    "grafana": func() (tfprotov5.ProviderServer, error) {
        // 1. Create mux server (SDKv2 + Framework, same as production)
        server, _ := provider.MakeProviderServer(ctx, "testacc")
        // 2. Get schema, build all-nil config (credentials come from env vars)
        schemaResp, _ := server.GetProviderSchema(ctx, nil)
        // 3. Configure provider with empty HCL config
        server.ConfigureProvider(ctx, &tfprotov5.ConfigureProviderRequest{Config: emptyConfig})
        return server, nil
    },
}
```

Provider credentials come from environment variables automatically (same `SetDefaults()` as production).

## Acceptance Test Pattern

```go
func TestAccFoo_basic(t *testing.T) {
    testutils.CheckOSSTestsEnabled(t)     // ← always first line; or CheckEnterpriseTestsEnabled, etc.
    // t.Parallel() — some tests can't be parallel due to global state

    var foo models.FooType

    resource.Test(t, resource.TestCase{
        ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
        CheckDestroy:             fooCheckExists.destroyed(&foo, nil),
        Steps: []resource.TestStep{
            {
                Config: testutils.TestAccExample(t, "resources/grafana_foo/_acc_basic.tf"),
                Check: resource.ComposeTestCheckFunc(
                    fooCheckExists.exists("grafana_foo.test", &foo),
                    resource.TestCheckResourceAttr("grafana_foo.test", "name", "expected"),
                    testutils.CheckLister("grafana_foo.test"),  // validates lister function
                ),
            },
            {
                // Update step
                Config: testutils.TestAccExample(t, "resources/grafana_foo/_acc_update.tf"),
                Check: resource.ComposeTestCheckFunc(...),
            },
            {
                // Import step
                ResourceName:      "grafana_foo.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## Example-Based Testing

`TestAccExample(t, "resources/grafana_foo/_acc_basic.tf")` loads:
```
internal/testutils/provider.go (runtime.Caller)
  └── ../../../examples/resources/grafana_foo/_acc_basic.tf
```

Example files in `examples/` serve **dual purpose**:
1. **Documentation source** — `tfplugindocs` generates `docs/` from them
2. **Acceptance test configs** — files prefixed with `_acc_` are loaded by `TestAccExample`

`TestAccExampleWithReplace(t, path, map[string]string)` replaces strings in the loaded HCL (useful for injecting unique names to avoid conflicts).

## CheckLister

`testutils.CheckLister("grafana_foo.test")` is a `resource.TestCheckFunc` that:
1. Gets the resource ID from Terraform state
2. Finds the resource's `ListIDsFunc` from the provider registry
3. Calls the lister against the live API
4. Asserts the resource ID appears in the returned list

Use this in at least one test step for every resource that has a lister function. It validates the code generation path is working.

## TestAccExamples (Mega-Test)

`internal/resources/examples_test.go:TestAccExamples` iterates ALL registered resources and data sources (including AppPlatform), matches each to its category (OSS/Enterprise/Cloud/etc.), loads the `resource.tf` or `datasource.tf` example, and runs it as an acceptance test. This ensures every resource has a working example.

## AppPlatform Unit Tests

AppPlatform resources (`internal/resources/appplatform/resource_test.go`) have **pure unit tests** that don't need a live Grafana instance:

```go
// No acceptance gate — always runs
func TestAppPlatformResourceSpec(t *testing.T) {
    // Table-driven
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            diags := parseSpec(ctx, tc.input, &result)
            require.False(t, diags.HasError())
            require.Equal(t, tc.expected, result)
        })
    }
}
```

These test `SpecParser`/`SpecSaver` functions using mock K8s objects and `testify/require`.

## Makefile Test Targets

| Target | What It Does |
|--------|-------------|
| `make testacc` | Build provider binary + `TF_ACC=1 go test ./...` |
| `make testacc-oss` | `TF_ACC_OSS=true` + testacc |
| `make testacc-enterprise` | `TF_ACC_ENTERPRISE=true` + testacc |
| `make testacc-cloud-api` | `TF_ACC_CLOUD_API=true` + testacc |
| `make testacc-cloud-instance` | `TF_ACC_CLOUD_INSTANCE=true` + testacc |
| `make testacc-oss-docker` | Docker Compose up + testacc-oss + Docker down |
| `make testacc-enterprise-docker` | Enterprise Docker image + testacc-enterprise |
| `make testacc-tls-docker` | mTLS proxy via ghostunnel + testacc-oss |
| `make testacc-subpath-docker` | Nginx subpath proxy + testacc-oss |
| `make integration-test` | Runs `testdata/integration/test.sh` |

Docker Compose services: `mysql` (Grafana DB), `grafana` (OSS or Enterprise), `mtls-proxy` (ghostunnel), `nginx` (subpath proxy).
