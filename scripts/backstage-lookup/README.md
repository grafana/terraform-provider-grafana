# backstage-lookup

Queries the EngHub (Backstage) catalog to find team ownership for Terraform
resources and assigns GitHub issues to the correct team project boards.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `BACKSTAGE_URL` | yes | Base URL of the Backstage instance |
| `IAP_AUDIENCE` | no | OAuth 2.0 client ID of the IAP-protected EngHub instance. When set, the tool authenticates through IAP using GCP Application Default Credentials. When unset, requests are made without authentication (for local dev via port-forward). |
| `GITHUB_TOKEN` | yes | GitHub token with the `project` scope |

## GCP prerequisites for IAP authentication

The following must be in place before the tool can authenticate to an
IAP-protected EngHub instance (e.g. from GitHub Actions):

### 1. Service account

A GCP service account that the tool will act as. In CI this is
`github-terraform-provider-ci@grafanalabs-workload-identity.iam.gserviceaccount.com`.

### 2. IAP access

The service account must have the `roles/iap.httpsResourceAccessor` IAM role
on the EngHub/Backstage IAP resource. This grants permission to pass through
the Identity-Aware Proxy.

Grant it with:

```console
gcloud iap web add-iam-policy-binding \
  --resource-type=backend-services \
  --service=<BACKSTAGE_BACKEND_SERVICE> \
  --member="serviceAccount:github-terraform-provider-ci@grafanalabs-workload-identity.iam.gserviceaccount.com" \
  --role="roles/iap.httpsResourceAccessor"
```

### 3. Workload Identity Federation (for GitHub Actions)

A Workload Identity Federation pool and provider must link GitHub Actions OIDC
tokens to the service account. The autoassign workflow is configured to use:

- Pool: `projects/304398677251/locations/global/workloadIdentityPools/github/providers/github-provider`
- Service account: `github-terraform-provider-ci@grafanalabs-workload-identity.iam.gserviceaccount.com`

The `google-github-actions/auth` action exchanges the GitHub OIDC token for
GCP credentials and sets up Application Default Credentials (ADC) in the
runner environment. The `idtoken` Go library then uses ADC to obtain an OIDC
ID token with the IAP audience, which is attached as
`Authorization: Bearer <id_token>` on every request to EngHub.

### 4. Vault secrets

The following secrets must be stored in Vault and accessible via the
`grafana/shared-workflows/actions/get-vault-secrets` action:

- `backstage:backstage_url` -- the EngHub/Backstage base URL
- `backstage:audience` -- the IAP OAuth 2.0 client ID (used as `IAP_AUDIENCE`)

## How it works

```
GitHub Actions OIDC token
  -> Workload Identity Federation (exchanges for GCP credentials)
    -> Application Default Credentials (ADC) in the environment
      -> idtoken.NewClient(ctx, IAP_AUDIENCE) obtains OIDC ID token
        -> Authorization: Bearer <id_token> on requests to EngHub
          -> IAP validates the token and forwards to Backstage
```

## Local development

### Option 1: Port-forward (no authentication)

Set up a port-forward to access Backstage directly, bypassing IAP:

```console
kubectl port-forward -n backstage service/backstage-ingress 8080
export BACKSTAGE_URL=http://localhost:8080
```

Get a GitHub token with the `project` scope:

```console
gh auth login -s 'project'
export GITHUB_TOKEN=$(gh auth token)
```

Run the tool (omit `IAP_AUDIENCE` to skip IAP auth):

```console
go run . <issueNumber> <resource1> [resource2] ...
```

### Option 2: Through IAP (authenticate with your Google account)

Log in with your Google account to set up ADC:

```console
gcloud auth application-default login
```

Set the environment variables:

```console
export BACKSTAGE_URL=<enghub-url>
export IAP_AUDIENCE=<iap-oauth-client-id>
export GITHUB_TOKEN=$(gh auth token)
```

Run the tool:

```console
go run . <issueNumber> <resource1> [resource2] ...
```

Note: your Google account must have `roles/iap.httpsResourceAccessor` on the
EngHub IAP resource for this to work.
