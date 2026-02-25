# Provider Mux Architecture

The Grafana Terraform Provider is a **muxed provider** — it combines HashiCorp SDKv2 and Plugin Framework into a single binary.

## Provider Assembly

```
main.go
  └── tf5server.Serve("registry.terraform.io/grafana/grafana", MakeProviderServer)
        │
        └── provider.MakeProviderServer(ctx, version)  [pkg/provider/provider.go]
              │
              ├── FrameworkProvider(version)             [framework_provider.go]
              │     implements provider.Provider (v6 protocol)
              │     ├── Resources():    pluginFrameworkResources() + appplatform
              │     ├── DataSources():  pluginFrameworkDataSources()
              │     └── Functions():    k6bundle
              │           │
              │           └── providerserver.NewProtocol6(FrameworkProvider)
              │                 │
              │                 └── tf6to5server.DowngradeServer(...)  [v6 → v5 adapter]
              │
              ├── Provider(version).GRPCProvider        [legacy_provider.go]
              │     implements schema.Provider (v5 protocol)
              │     ├── ResourcesMap:     legacySDKResources()
              │     └── DataSourcesMap:  legacySDKDataSources()
              │
              └── tf5muxserver.NewMuxServer(ctx, [downgraded_v6, native_v5])
                    Routes each resource/datasource type name to correct sub-provider
```

**Why the downgrade?** The mux requires uniform protocol (v5). FrameworkProvider natively speaks v6; `tf6to5server.DowngradeServer` wraps it in a translation layer so both can be muxed together.

## Resource Registration Flow

All resources are aggregated in `pkg/provider/resources.go`:

```
Resources() []*common.Resource
  ← cloud.Resources
  ← grafana.Resources
  ← oncall.Resources
  ← machinelearning.Resources
  ← slo.Resources
  ← k6.Resources
  ← syntheticmonitoring.Resources
  ← cloudprovider.Resources
  ← connections.Resources
  ← fleetmanagement.Resources
  ← frontendo11y.Resources
  ← asserts.Resources

Then routed:
  legacySDKResources()       → r.Schema != nil        → SDKv2 provider
  pluginFrameworkResources() → r.PluginFrameworkSchema != nil → Framework provider

AppPlatformResources() []appplatform.NamedResource   ← completely separate
  ← appplatform.Dashboard()
  ← appplatform.Playlist()
  ← ... (11 total)
  → Framework provider only (bypass *common.Resource entirely)
```

## Configuration Flow

```
HCL terraform block
  └── ProviderConfig struct     [framework_provider.go:22]
        (decoded by tfprotov5 via tfsdk tags)
        │
        └── cfg.SetDefaults()   — applies env var fallbacks
              ├── GRAFANA_URL    → cfg.URL
              ├── GRAFANA_AUTH   → cfg.Auth
              ├── GRAFANA_ORG_ID → cfg.OrgID
              └── ... (30+ env vars total)
              │
              └── CreateClients(cfg)   [configure_clients.go]
                    Returns *common.Client with all API clients
```

Both sub-providers call `CreateClients` independently — they each have their own `*common.Client` instance. The two instances are equivalent (same config, same env vars) but are **not shared**. Mutexes on one instance don't protect the other, but this is safe because each resource type is registered in exactly one sub-provider.

## ProviderConfig Fields

Full struct in `pkg/provider/framework_provider.go:22`. Key groups:

| Group | Fields |
|-------|--------|
| Grafana core | `URL`, `Auth`, `OrgID`, `StackID`, `HTTPHeaders`, `Retries`, `RetryStatusCodes`, `RetryWait` |
| TLS | `TLSKey`, `TLSCert`, `CACert`, `InsecureSkipVerify` |
| Dashboard | `StoreDashboardSha256` |
| Grafana Cloud | `CloudAccessPolicyToken`, `CloudAPIURL` |
| Synthetic Monitoring | `SMAccessToken`, `SMURL` |
| OnCall | `OncallAccessToken`, `OncallURL` |
| Cloud Provider | `CloudProviderAccessToken`, `CloudProviderURL` |
| Connections | `ConnectionsAPIAccessToken`, `ConnectionsAPIURL` |
| Fleet Management | `FleetManagementAuth`, `FleetManagementURL` |
| Frontend O11y | `FrontendO11YAPIURL`, `FrontendO11yAPIAccessToken` |
| k6 | `K6AccessToken` |
| Asserts | `AssertsAPIURL`, `AssertsAPIKey` |

Fields with `tfsdk:"-"` are not in the HCL schema: `UserAgent`, `Version`.

## How Resources Are Disambiguated

When Terraform calls a resource operation, the mux routes by type name:
- `grafana_dashboard` → SDKv2 sub-provider (`.Schema != nil`)
- `grafana_k6_project` → Framework sub-provider (`.PluginFrameworkSchema != nil`)
- `grafana_apps_dashboard_dashboard_v1beta1` → Framework sub-provider (registered directly by `AppPlatformResources()`)

A resource can be in **exactly one** sub-provider. `ResourceCommon` supports both fields, but only one should be set per resource.
