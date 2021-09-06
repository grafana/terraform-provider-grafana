<a href="https://terraform.io">
  <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" title="Terraform" align="right" height="50" />
</a>

# Terraform Provider for Grafana

[![Build Status](https://drone.grafana.net/api/badges/grafana/terraform-provider-grafana/status.svg)](https://drone.grafana.net/grafana/terraform-provider-grafana)

- Grafana website: https://grafana.com
* Grafana Cloud website: https://grafana.com/products/cloud/
- Provider Documentation: https://registry.terraform.io/providers/grafana/grafana/latest/docs
- Grafana Chat: [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT)

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) 0.12+

## Development

If you're new to provider development, a good place to start is the [Extending
Terraform](https://www.terraform.io/docs/extend/index.html) docs.

Set up your local environment by installing [Go](http://www.golang.org). Also
[Docker](https://docs.docker.com/install/) can be used for running tests.

### Running Tests

Acceptance tests require a running instance of Grafana. You can either handle
running an instance of Grafana yourself or use `docker-compose`.

If you choose `docker-compose`, run `make testacc-docker`. This is the simplest
option, but often not the quickest.

Alternatively you can use the `testacc` target which will use your local `go`
installation:

```sh
# Assuming Grafana was run with:
# docker run --rm -p 3000:3000 grafana/grafana

GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
GRAFANA_ORG_ID=1 \
make testacc
```

#### Running enterprise tests

To run tests for resources which are available only for Grafana Enterprise, running instance of Grafana Enterprise is required.
It is only possible to run tests for Grafana Enterprise using local environment.

```sh
# Assuming Grafana was run with:
# docker run --rm -p 3000:3000 grafana/grafana

GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
GRAFANA_ORG_ID=1 \
make testacc-enterprise
```

## Documentation

Documentation is generated with
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). Generated
files are in `docs/` and should not be updated manually. They are derived from:

* Schema `Description` fields in the provider Go code.
* [examples/](./examples)
* [templates/](./templates)

Use `go generate` to update generated docs.

## Releasing

Builds and releases are automated with GitHub Actions and
[GoReleaser](https://github.com/goreleaser/goreleaser/). The changelog is
managed with
[github-changelog-generator](https://github.com/github-changelog-generator/github-changelog-generator).

Currently there are a few manual steps to this:

1. Update the changelog:

   ```sh
   RELEASE_VERSION=v... \
   CHANGELOG_GITHUB_TOKEN=... \
   make changelog
   ```

   This will commit the changelog locally.

2. Review generated changelog and push:

   View the committed changelog with `git show`. If all is well `git push origin
   master`.

3. Kick off the release:

   ```sh
   RELEASE_VERSION=v... \
   make release
   ```

   Once the command exits, you can monitor the rest of the process on the
   [Actions
   UI](https://github.com/grafana/terraform-provider-grafana/actions?query=workflow%3Arelease).

4. Publish release:

   The Action creates the release, but leaves it in "draft" state. Open it up in
   a [browser](https://github.com/grafana/terraform-provider-grafana/releases)
   and if all looks well, mash the publish button.
