# Git Sync Terraform Resources — Implementation Plan

**Prerequisite:** Generic secure block support (see `secure_block_support.md`).

## Overview

Implement Terraform resources for configuring Git Sync via the Grafana provisioning app platform.
The provisioning app (`provisioning.grafana.app/v0alpha1`) exposes two user-facing CRDs:

| Resource | Terraform Name | Purpose |
|---|---|---|
| Repository | `grafana_apps_provisioning_repository_v0alpha1` | Configures a git repository for syncing Grafana resources |
| Connection | `grafana_apps_provisioning_connection_v0alpha1` | Configures provider credentials (GitHub App) used by repositories |

### Resources NOT Implemented (and why)

| Resource | Reason |
|---|---|
| Job | Operational/imperative — represents a one-time sync action, not declarative config |
| HistoricJob | Internal append-only log — read-only, no user provisioning use case |

---

## Architecture Decision: Local Wrapper Objects + Upstream API Types

The provisioning app does **not** expose app-sdk `Kind` helpers in Grafana, so the Terraform
provider still defines local wrapper objects (`ProvisioningRepository`, `ProvisioningConnection`)
that implement `sdkresource.Object` / `sdkresource.ListObject`.

To reduce duplication, these wrappers reuse upstream provisioning API types wherever possible:

- spec/secure structs and enums are imported from
  `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1`
- Terraform model structs remain provider-local
- `RepositoryKind()` / `ConnectionKind()` and codecs remain provider-local
- Secure subresource accessors are generic via `secureSubresourceSupport[T]`
  (`internal/resources/appplatform/secure_subresource_support.go`)

This keeps provider wiring local while avoiding re-defining upstream API shape/constants.

---

## File Structure

```
internal/resources/appplatform/
  repository_resource.go       # Repository types, Kind, codec, TF resource definition
  connection_resource.go       # Connection types, Kind, codec, TF resource definition

examples/resources/
  grafana_apps_provisioning_repository_v0alpha1/
    resource.tf                # GitHub repository example
  grafana_apps_provisioning_connection_v0alpha1/
    resource.tf                # GitHub App connection example
```

Registration in `pkg/provider/resources.go`:
```go
func AppPlatformResources() []appplatform.NamedResource {
    return []appplatform.NamedResource{
        // ... existing resources ...
        appplatform.Repository(),
        appplatform.Connection(),
    }
}
```

---

## Resource 1: Repository (`grafana_apps_provisioning_repository_v0alpha1`)

### Source Types (from `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1/types.go`)

```go
type RepositorySpec struct {
    Title       string                    `json:"title"`
    Description string                    `json:"description,omitempty"`
    Workflows   []Workflow                `json:"workflows"`
    Sync        SyncOptions               `json:"sync"`
    Type        RepositoryType            `json:"type"`
    Local       *LocalRepositoryConfig    `json:"local,omitempty"`
    GitHub      *GitHubRepositoryConfig   `json:"github,omitempty"`
    Git         *GitRepositoryConfig      `json:"git,omitempty"`
    Bitbucket   *BitbucketRepositoryConfig`json:"bitbucket,omitempty"`
    GitLab      *GitLabRepositoryConfig   `json:"gitlab,omitempty"`
    Connection  *ConnectionInfo           `json:"connection,omitempty"`
}

type SecureValues struct {
    Token         InlineSecureValue `json:"token,omitzero,omitempty"`
    WebhookSecret InlineSecureValue `json:"webhookSecret,omitzero,omitempty"`
}
```

### Terraform Schema

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "example" {
  metadata {
    uid = "my-github-repo"
  }

  spec {
    title       = "My GitHub Repository"
    description = "Syncs dashboards from GitHub"
    type        = "github"

    workflows = ["write"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    # Exactly one of these blocks must be provided, matching `type`:

    github {
      url                          = "https://github.com/example/test"
      branch                       = "main"
      path                         = "grafana/"
      generate_dashboard_previews  = false
    }

    # git {
    #   url    = "https://example.com/repo.git"
    #   branch = "main"
    #   path   = "grafana/"
    # }

    # bitbucket {
    #   url        = "https://bitbucket.org/example/test"
    #   branch     = "main"
    #   token_user = "x-token-auth"
    #   path       = "grafana/"
    # }

    # gitlab {
    #   url    = "https://gitlab.com/example/test"
    #   branch = "main"
    #   path   = "grafana/"
    # }

    # local {
    #   path = "/var/lib/grafana/dashboards"
    # }

    # connection {
    #   name = "my-github-connection"
    # }
  }

  secure {
    token = {
      create = var.github_token
    }
    webhook_secret = {
      create = var.webhook_secret
    }
  }
  secure_version = 1
}
```

### Attribute Details

**spec block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `title` | string | Required | Display name shown in the UI |
| `description` | string | Optional | Repository description |
| `type` | string | Required | One of: `local`, `github`, `git`, `bitbucket`, `gitlab` |
| `workflows` | list(string) | Optional | Allowed change workflows: `write`, `branch`. Empty = read-only |
| `sync` | block | Required | Sync configuration |
| `github` | block | Optional | GitHub-specific config (when type=github) |
| `git` | block | Optional | Generic git config (when type=git) |
| `bitbucket` | block | Optional | Bitbucket-specific config (when type=bitbucket) |
| `gitlab` | block | Optional | GitLab-specific config (when type=gitlab) |
| `local` | block | Optional | Local filesystem config (when type=local) |
| `connection` | block | Optional | Connection reference (for Connection-based auth) |

**spec.sync block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `enabled` | bool | Required | Whether sync is enabled |
| `target` | string | Required | `instance` (global) or `folder` (managed folder) |
| `interval_seconds` | number | Optional | Sync interval (system defines minimum) |

**spec.github block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `url` | string | Required | Repository URL (e.g. `https://github.com/org/repo`) |
| `branch` | string | Required | Branch to sync |
| `path` | string | Optional | Subdirectory for Grafana data (e.g. `grafana/`) |
| `generate_dashboard_previews` | bool | Optional | Show dashboard previews for PRs |

**spec.git block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `url` | string | Required | Repository URL (e.g. `https://example.com/repo.git`) |
| `branch` | string | Required | Branch to sync |
| `token_user` | string | Optional | Username for HTTP Basic Auth with PAT (defaults to `git`) |
| `path` | string | Optional | Subdirectory for Grafana data |

**spec.bitbucket block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `url` | string | Required | Repository URL |
| `branch` | string | Required | Branch to sync |
| `token_user` | string | Optional | Username for HTTP Basic Auth — use `x-token-auth` for Bitbucket access tokens (defaults to `git`, which won't work with Bitbucket) |
| `path` | string | Optional | Subdirectory for Grafana data |

**spec.gitlab block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `url` | string | Required | Repository URL |
| `branch` | string | Required | Branch to sync |
| `path` | string | Optional | Subdirectory for Grafana data |

**spec.local block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `path` | string | Required | Filesystem path |

**spec.connection block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `name` | string | Required | Name of the Connection resource to use |

**secure block** (all attributes are WriteOnly — never stored in state):

| Attribute | Type | Required | Description |
|---|---|---|---|
| `token` | object `{ create?, name? }` | Optional | Token for repository auth (`create` inline value or `name` existing secret reference) |
| `webhook_secret` | object `{ create?, name? }` | Optional | Webhook secret (`create` inline value or `name` existing secret reference) |

---

## Resource 2: Connection (`grafana_apps_provisioning_connection_v0alpha1`)

### Source Types (from `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1/connections.go`)

```go
type ConnectionSpec struct {
    Title       string                    `json:"title"`
    Description string                    `json:"description,omitempty"`
    Type        ConnectionType            `json:"type"`
    URL         string                    `json:"url,omitempty"`
    GitHub      *GitHubConnectionConfig   `json:"github,omitempty"`
}

type ConnectionSecure struct {
    PrivateKey   InlineSecureValue `json:"privateKey,omitzero,omitempty"`
    ClientSecret InlineSecureValue `json:"clientSecret,omitzero,omitempty"`
    Token        InlineSecureValue `json:"token,omitzero,omitempty"`
}
```

### Terraform Schema

```hcl
resource "grafana_apps_provisioning_connection_v0alpha1" "example" {
  metadata {
    uid = "my-github-connection"
  }

  spec {
    title       = "My GitHub App Connection"
    description = "GitHub App for provisioning"
    type        = "github"
    url         = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = file("path/to/private-key.pem")
    }
    client_secret = {
      create = var.client_secret
    }
    token = {
      create = var.token
    }
  }
  secure_version = 1
}
```

### Attribute Details

**spec block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `title` | string | Required | Display name shown in the UI |
| `description` | string | Optional | Connection description |
| `type` | string | Required | Provider type: `github` |
| `url` | string | Optional | Connection URL |
| `github` | block | Optional | GitHub App config (when type=github) |

**spec.github block:**

| Attribute | Type | Required | Description |
|---|---|---|---|
| `app_id` | string | Required | GitHub App ID |
| `installation_id` | string | Required | GitHub App Installation ID |

**secure block** (all attributes are WriteOnly — never stored in state):

| Attribute | Type | Required | Description |
|---|---|---|---|
| `private_key` | object `{ create?, name? }` | Optional | Private key for GitHub App auth (`create` inline value or `name` existing secret reference) |
| `client_secret` | object `{ create?, name? }` | Optional | Client secret (`create` inline value or `name` existing secret reference) |
| `token` | object `{ create?, name? }` | Optional | Token for auth (`create` inline value or `name` existing secret reference) |

---

## Implementation Steps

### Step 1: Implement Repository Resource (`repository_resource.go`)

1. Define local types implementing `sdkresource.Object`:
   - `ProvisioningRepository`, `ProvisioningRepositoryList`
   - Reuse upstream `v0alpha1` spec/secure enums and structs via type aliases
   - `RepositoryKind()` + JSON codec

2. Define Terraform models:
   - `RepositorySpecModel` with nested struct models for each provider type

3. Implement `Repository()` function returning `NamedResource` with
   `SecureValueAttributes` + `SecureParser: DefaultSecureParser[*ProvisioningRepository]`

4. Implement `SpecParser`, `SpecSaver`, and `SecureParser` functions

### Step 2: Implement Connection Resource (`connection_resource.go`)

1. Define local types implementing `sdkresource.Object`:
   - `ProvisioningConnection`, `ProvisioningConnectionList`
   - Reuse upstream `v0alpha1` spec/secure structs via type aliases
   - `ConnectionKind()` + JSON codec

2. Define Terraform models:
   - `ConnectionSpecModel` with nested provider config models

3. Implement `Connection()` function returning `NamedResource` with
   `SecureValueAttributes` + `SecureParser: DefaultSecureParser[*ProvisioningConnection]`

4. Implement `SpecParser`, `SpecSaver`, and `SecureParser` functions

### Step 3: Register Resources

Add to `pkg/provider/resources.go`:
```go
func AppPlatformResources() []appplatform.NamedResource {
    return []appplatform.NamedResource{
        // ... existing ...
        appplatform.Repository(),
        appplatform.Connection(),
    }
}
```

### Step 4: Create Example Files

### Step 5: Add Acceptance Tests

Follow pattern from `alertenrichment_resource_acc_test.go`.

### Step 6: Validate with Dockerized Local Harness

Use the local harness in `test-local/git-sync` for fast regression checks against nightly Grafana:

```bash
./test-local/git-sync/up.sh

./test-local/git-sync/run-example.sh 01_connection_github_app apply
./test-local/git-sync/run-example.sh 02_repository_github_token apply
./test-local/git-sync/run-example.sh 03_repository_with_connection apply

./test-local/git-sync/run-example.sh 03_repository_with_connection destroy
./test-local/git-sync/run-example.sh 02_repository_github_token destroy
./test-local/git-sync/run-example.sh 01_connection_github_app destroy

./test-local/git-sync/down.sh
```

Known behavior to account for:
- `03_repository_with_connection` destroy can require one immediate retry because the API may still
  consider the just-deleted repository as a transient reference to the connection.

---

## Example Terraform Configurations

### Example 1: GitHub Repository with Token Auth

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "github_repo" {
  metadata {
    uid = "my-github-repo"
  }

  spec {
    title = "My GitHub Repository"
    type  = "github"

    workflows = ["write"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    github {
      url    = "https://github.com/my-org/grafana-dashboards"
      branch = "main"
      path   = "grafana/"
    }
  }

  secure {
    token = {
      create = var.github_token
    }
  }
  secure_version = 1
}
```

### Example 2: Pure Git Repository (any Git server)

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "pure_git" {
  metadata {
    uid = "my-pure-git-repo"
  }

  spec {
    title = "Self-Hosted Git Server"
    type  = "git"

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 120
    }

    git {
      url    = "https://git.internal.example.com/infra/grafana-dashboards.git"
      branch = "main"
      path   = "grafana/"
    }
  }

  secure {
    token = {
      create = var.git_pat
    }
  }
  secure_version = 1
}
```

### Example 3: GitHub Repository with Connection Auth (GitHub App)

```hcl
resource "grafana_apps_provisioning_connection_v0alpha1" "github_app" {
  metadata {
    uid = "github-app-connection"
  }

  spec {
    title = "GitHub App"
    type  = "github"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = file("${path.module}/github-app-private-key.pem")
    }
  }
  secure_version = 1
}

resource "grafana_apps_provisioning_repository_v0alpha1" "github_repo_via_app" {
  metadata {
    uid = "dashboards-repo"
  }

  spec {
    title = "Dashboards Repository"
    type  = "github"

    workflows = ["branch"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 300
    }

    github {
      url                         = "https://github.com/my-org/grafana-dashboards"
      branch                      = "main"
      path                        = "grafana/"
      generate_dashboard_previews = true
    }

    connection {
      name = grafana_apps_provisioning_connection_v0alpha1.github_app.metadata[0].uid
    }
  }
}
```

### Example 4: GitLab Repository

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "gitlab_repo" {
  metadata {
    uid = "my-gitlab-repo"
  }

  spec {
    title = "GitLab Dashboards"
    type  = "gitlab"

    workflows = ["write", "branch"]

    sync {
      enabled = true
      target  = "folder"
    }

    gitlab {
      url    = "https://gitlab.com/my-group/grafana-dashboards"
      branch = "main"
      path   = "grafana/"
    }
  }

  secure {
    token = {
      create = var.gitlab_token
    }
  }
  secure_version = 1
}
```

### Example 5: Bitbucket Repository

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "bitbucket_repo" {
  metadata {
    uid = "my-bitbucket-repo"
  }

  spec {
    title = "Bitbucket Dashboards"
    type  = "bitbucket"

    sync {
      enabled = true
      target  = "folder"
    }

    bitbucket {
      url        = "https://bitbucket.org/my-workspace/grafana-dashboards"
      branch     = "main"
      path       = "grafana/"
      token_user = "x-token-auth"  # Required for Bitbucket access tokens (default "git" won't work)
    }
  }

  secure {
    token = {
      create = var.bitbucket_token
    }
  }
  secure_version = 1
}
```

---

## Source File References

| File | Purpose |
|---|---|
| `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1/types.go` | Repository, RepositorySpec, SecureValues, SyncOptions (hand-written, source of truth) |
| `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1/connections.go` | Connection, ConnectionSpec, ConnectionSecure (hand-written, source of truth) |
| `grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1/register.go` | API group constants (`provisioning.grafana.app`, `v0alpha1`) |
| `grafana/apps/provisioning/pkg/repository/git/repository.go` | TokenUser default logic (`"git"` fallback) |
| `terraform-provider-grafana/internal/resources/appplatform/resource.go` | Generic Resource[T, L] framework |
| `terraform-provider-grafana/internal/resources/appplatform/appo11y_config_resource.go` | Reference for self-contained type pattern |
