# Contributing

We welcome contributions. Here’s how to get changes merged.

## Submitting changes

1. **Fork the repo** and create a branch from `main`.
2. **Make your changes.** Follow the existing code style and patterns.
3. **Run tests** for the code you touch:
   - Unit tests: `go test ./...`
   - OSS acceptance tests (needs Grafana): see [README – Running Tests](README.md#running-tests), e.g. `make testacc-oss-docker`.
4. **Run the linter** before opening a PR: `make golangci-lint` (runs `golangci-lint run ./... -v` in Docker with the same version CI uses). If you have [golangci-lint](https://golangci-lint.run/) v2 installed locally, you can run `golangci-lint run ./... -v` from the repo root instead.
5. **Update generated docs** if you changed resource/datasource schema or examples: run `go generate ./...` (or `make docs`). CI will fail if `docs/` is out of sync.
6. **Open a pull request** against `main` — see [PR title format](#pr-title-format) below.

## PR title format

This repository uses **squash merges**, so all commits in a PR are combined into a single commit when merged. The **PR title becomes the commit message**, so it must follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format:

```
<type>(<scope>): <subject>
```

A CI check validates the PR title and will block merging if it doesn't conform.

### Types

| Type | Purpose |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, no logic change |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `build` | Build system or external dependencies |
| `ci` | CI/CD configuration |
| `chore` | Maintenance tasks, dependency updates, housekeeping |

### Scope

Scope is optional but recommended. Use the affected resource name when applicable. Scopes must be lowercase with no spaces.

### Subject

The subject should be lowercase, use imperative mood ("add" not "added"), and not end with a period.

### Breaking changes

Breaking changes should be avoided whenever possible. Terraform users depend on stable resource schemas, and breaking changes force manual state manipulation or configuration rewrites. Prefer deprecating attributes over removing them, and add new attributes alongside old ones when feasible.

When a breaking change is unavoidable, append `!` after the type (and scope, if present):

```
feat(grafana_folder)!: rename uid to folder_uid
```

Breaking changes must be explained in the PR description body. When squash merging, expand the commit message body to include a `BREAKING CHANGE:` footer describing what changed and how users should migrate. For example:

```
feat(grafana_folder)!: rename uid to folder_uid

BREAKING CHANGE: The `uid` attribute on `grafana_folder` has been renamed
to `folder_uid`. Update your configuration and state accordingly.
```

### Examples

- `feat(grafana_dashboard): add uid attribute`
- `fix(grafana_folder): handle missing parent on import`
- `refactor(grafana_team): migrate to Plugin Framework`
- `feat(grafana_folder)!: rename uid to folder_uid`
- `chore: update Go dependencies`
- `docs: regenerate resource documentation`

## Questions

For questions or discussion, use the [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT). The Grafana Slack is public—anyone can join at [slack.grafana.com](https://slack.grafana.com/).
