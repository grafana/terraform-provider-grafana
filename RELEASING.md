# Releasing

## Versioning

This project follows [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`).
Version bumps are determined automatically from
[conventional commit](https://www.conventionalcommits.org/) messages using
[git-cliff](https://git-cliff.org).

- **Major** (`vX.0.0`): Reserved for intentional breaking changes. Major releases
  should be rare and well-documented with an upgrade guide. Examples:
  - Removing or renaming existing resources or data sources
  - Removing or renaming attributes
  - Changing attribute types (e.g. string to list)
  - Changing provider configuration in incompatible ways
  - Any change that requires users to rewrite their Terraform configuration or
    manually manipulate state

- **Minor** (`vX.Y.0`): New features and improvements that are
  backwards-compatible. **This is the most common release type.** Examples:
  - Adding new resources or data sources
  - Adding new optional attributes to existing resources
  - Refactoring internals (e.g. migrating a resource from SDKv2 to Plugin Framework)
  - Performance improvements
  - Removing resources that are clearly marked as alpha, beta, or experimental
    (these carry no stability guarantee)

- **Patch** (`vX.Y.Z`): Backwards-compatible bug fixes and dependency updates.
  These are typically follow-up releases after a minor release. Examples:
  - Fixing incorrect API calls or state handling
  - Correcting plan-time validation logic
  - Fixing documentation typos in schema `Description` fields
  - Go dependency updates (security patches, bug fixes)
  - Other small, targeted fixes

### Bump rules

The version bump is computed by git-cliff from the conventional commit messages
since the last tag. The rules are configured in [`cliff.toml`](cliff.toml) under
the `[bump]` section.

| Commit type(s) since last tag                             | Version bump          |
|-----------------------------------------------------------|-----------------------|
| Only `fix` (including `fix(deps)`)                        | **patch** (vX.Y.Z)   |
| `feat`, `refactor`, or `perf` (with or without others)    | **minor** (vX.Y.0)   |
| `BREAKING CHANGE` footer or `!` suffix on any commit type | **major** (vX.0.0)   |
| Only `ci`, `docs`, `test`, `build`, `style`, `chore` | No bump (nothing to release) |

### Commit types and release impact

| Commit type | Affects binary? | Triggers release? | Bump    |
|-------------|-----------------|-------------------|---------|
| `feat`      | yes             | yes               | minor   |
| `fix`       | yes             | yes               | patch   |
| `refactor`  | yes             | yes               | minor   |
| `perf`      | yes             | yes               | minor   |
| `docs`      | no              | no                | —       |
| `ci`        | no              | no                | —       |
| `test`      | no              | no                | —       |
| `build`     | no              | no                | —       |
| `style`     | no              | no                | —       |
| `chore`     | no              | no                | —       |

### Dependency updates

Go module dependency updates are managed by Renovate and use `fix(deps)` as
their commit type, because changes to `go.mod`/`go.sum` affect the compiled
provider binary. These trigger a **patch** release.

GitHub Actions and other non-Go dependency updates use `chore(deps)` and do
**not** trigger a release, since they have no effect on the shipped binary.

This behavior is configured in [`.github/renovate.json5`](.github/renovate.json5).

## Creating a release

### Prerequisites

- [git-cliff](https://git-cliff.org/docs/installation) installed locally
- Push access to the repository

### Steps

Before tagging, it is **strongly recommended** to validate the
unpublished build against the internal `tfprovidertest` appenv stack —
see [Validating unpublished builds](#validating-unpublished-builds).

1. Switch to the `main` branch:

   ```sh
   git checkout main
   ```

2. Run the release target:

   ```sh
   make release
   ```

   The target will:
   - Verify you are on `main` (refuses to run from other branches)
   - Fetch from origin and prompt to pull if your local branch is behind
   - Compute the next version from conventional commits using git-cliff

   To override the computed version:

   ```sh
   RELEASE_VERSION=v5.0.0 make release
   ```

3. The tag push triggers the [`release`](.github/workflows/release.yml) GitHub
   Actions workflow (see [CI pipeline](#ci-pipeline) below).

4. Open the [Releases page](https://github.com/grafana/terraform-provider-grafana/releases),
   review the draft release, and publish it.

### What `make release` does

`make release` runs [`scripts/release.sh`](scripts/release.sh), which performs
the following checks and actions:

```
scripts/release.sh
  │
  ├─ on main branch?       → error if not
  ├─ fetch origin/main
  │   └─ local behind?     → prompt to pull (--ff-only)
  │
  ├─ RELEASE_VERSION set?  → use it
  ├─ git-cliff installed?  → compute version via `git cliff --bumped-version`
  │   └─ version == latest tag?  → error: nothing to release
  └─ neither?              → error: install git-cliff or set RELEASE_VERSION
  │
  ├─ git tag <version>
  └─ git push origin <version>
```

## CI pipeline

The [`release.yml`](.github/workflows/release.yml) workflow triggers on any tag
push matching `v*`. It runs the following steps:

1. **Generate changelog** — git-cliff produces release notes from conventional
   commits since the previous tag (`--latest --strip header`).

2. **Validate semver bump** — git-cliff computes the minimum required version
   (`--bumped-version`) and compares it against the pushed tag. If the tag is
   lower than what the commits require (e.g., tagging a patch when there are
   `feat` commits), the workflow **fails** before building. This is a safety net
   that catches incorrect manual overrides.

3. **Build and sign** — [GoReleaser](https://goreleaser.com/) builds the
   provider binary for all supported platforms (`linux`, `darwin`, `windows`,
   `freebsd` on `amd64`, `arm64`, `arm`, `386`), creates a separate
   `terraform-provider-grafana-generate` tool, computes SHA256 checksums, and
   signs them with GPG.

4. **Create draft release** — GoReleaser creates a GitHub Release in **draft**
   state with the git-cliff changelog as release notes. Someone with release
   access must manually review and publish the release.

### Validating unpublished builds

The [`validate-unpublished-provider.yml`](.github/workflows/validate-unpublished-provider.yml)
workflow is an optional (but strongly recommended), manual pre-release check.

#### How to validate

Run the **validate unpublished provider** GitHub Actions workflow for this [repository](https://github.com/grafana/terraform-provider-grafana/actions). No inputs are required.

#### How it works

1. **Build** — compiles a Linux `amd64` provider binary and uploads it as artifact
   `terraform-provider-grafana_linux_amd64`.
2. **Validate** — dispatches field-eng
   [`Deploy AppEnv`](https://github.com/grafana/field-eng-appenv-deployment/actions/workflows/generic_deploy.yml)
   for `tfprovidertest` with `grafana_provider_dev_override_run_id` set to this run’s ID,
   and waits for the deploy to finish (field-eng downloads the artifact and uses Terraform
   `dev_overrides` instead of the registry provider).

Script: [`validate-unpublished-provider-dispatch-and-wait.sh`](scripts/validate-unpublished-provider-dispatch-and-wait.sh).
Credentials: GATB via `create-github-app-token` (app `terraform-provider-grafana`); requires a matching entry in `deployment_tools` `github-app-configs/config.yaml`.

### Configuration files

| File                                  | Purpose                                    |
|---------------------------------------|--------------------------------------------|
| [`scripts/release.sh`](scripts/release.sh) | Local release script (branch check, pull prompt, version computation, tag + push) |
| [`cliff.toml`](cliff.toml)           | Changelog format, commit parsing, bump rules |
| [`.goreleaser.yml`](.goreleaser.yml)  | Build targets, archives, signing, release settings |
| [`.github/workflows/release.yml`](.github/workflows/release.yml) | CI workflow that orchestrates the release |
| [`.github/workflows/validate-unpublished-provider.yml`](.github/workflows/validate-unpublished-provider.yml) | Optional manual pre-release validation against field-eng appenv deploy |
| [`.github/renovate.json5`](.github/renovate.json5) | Renovate config for dependency update commit types |
