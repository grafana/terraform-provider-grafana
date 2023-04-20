<a href="https://terraform.io">
  <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" title="Terraform" align="right" height="50" />
</a>

# Terraform Provider for Grafana

[![Build Status](https://drone.grafana.net/api/badges/grafana/terraform-provider-grafana/status.svg)](https://drone.grafana.net/grafana/terraform-provider-grafana)

- Grafana website: <https://grafana.com>
- Grafana Cloud website: <https://grafana.com/products/cloud/>
- Provider Documentation: <https://registry.terraform.io/providers/grafana/grafana/latest/docs>
- Grafana Chat: [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12+

## Development

If you're new to provider development, a good place to start is the [Extending
Terraform](https://www.terraform.io/docs/extend/index.html) docs.

Set up your local environment by installing [Go](http://www.golang.org). Also
[Docker](https://docs.docker.com/install/) can be used for running tests.

## Local Development with Grafana
If you develop the provider and want to test locally with your grafana provider
1. create a `.terraformrc` and paste the following
```
provider_installation {
   dev_overrides {
      "grafana/grafana" = "/path/to/your/grafana/terraform-provider" # this path is the diretory where the binary is built
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
2. Run `go build` in this directory to get the binary, Terraform will use the binary you just built (it should print out a warning)

## Testing the `grafana-api-golang-client` Together with the Terraform Provider
As for testing the client, make a branch and open the provider PR with a `TODO:` to remove the replace operator, because the best way to test the changes you just made in the client is to integrate everything in the provider

1. create a branch `api-client-branchname` with your changes
2. modify the provider `go.mod`: 
 - replace github.com/grafana/grafana-api-golang-client => github.com/grafana/grafana-api-golang-client <api-client-branchname>
3. run `go mod tidy` in this directory
4. pushing allows you to run the provider tests in CI

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

- Schema `Description` fields in the provider Go code.
- [examples/](./examples)
- [templates/](./templates)

Use `go generate` to update generated docs.

## Releasing

Builds and releases are automated with GitHub Actions and
[GoReleaser](https://github.com/goreleaser/goreleaser/).

Currently there are a few manual steps to this:

1. Kick off the release:

   ```sh
   RELEASE_VERSION=v... \
   make release
   ```

2. Publish release:

   The Action creates the release, but leaves it in "draft" state. Open it up in
   a [browser](https://github.com/grafana/terraform-provider-grafana/releases)
   and if all looks well, click the `Auto-generate release notes` button and mash the publish button.
