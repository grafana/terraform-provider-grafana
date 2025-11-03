<a href="https://terraform.io">
  <img src="https://www.datocms-assets.com/2885/1629941242-logo-terraform-main.svg" title="Terraform" align="right" height="40px" />
</a>

# Terraform Provider for Grafana

[![Acceptance Tests](https://github.com/grafana/terraform-provider-grafana/actions/workflows/acc-tests.yml/badge.svg)](https://github.com/grafana/terraform-provider-grafana/actions/workflows/acc-tests.yml)
[![Unit Tests](https://github.com/grafana/terraform-provider-grafana/actions/workflows/unit-tests.yml/badge.svg)](https://github.com/grafana/terraform-provider-grafana/actions/workflows/unit-tests.yml)
[![Cloud Acceptance Tests](https://github.com/grafana/terraform-provider-grafana/actions/workflows/cloud-acc-tests.yml/badge.svg)](https://github.com/grafana/terraform-provider-grafana/actions/workflows/cloud-acc-tests.yml)

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
If you develop the provider and want to test locally with your Grafana provider
1. Create a `.terraformrc` file in your operating system user directory and paste the following
```
provider_installation {
   dev_overrides {
      "grafana/grafana" = "/path/to/your/terraform-provider-grafana" # this path is the directory where the binary is built
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
2. Run `go build` in this directory to get the binary, Terraform will use the binary you just built for every terraform plan/apply (it should print out a warning). No need to run terraform init.

### Running Tests

Acceptance tests require a running instance of Grafana. You can either handle
running an instance of Grafana yourself or use Docker Compose.

There are Make targets provided for different test scenarios that will start Grafana in Docker Compose automatically.
This is the simplest option, but often not the quickest.

To run the OSS tests with Grafana running in Docker Compose:

```sh
make testacc-oss-docker
```

Alternatively, you can use the `testacc` target which will use your local `go`
installation:

```sh
# Assuming Grafana was run with:
# docker run --rm -p 3000:3000 grafana/grafana

GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
make testacc
```

#### Running enterprise tests

To run tests for resources which are available only for Grafana Enterprise, running instance of Grafana Enterprise is required.
It is only possible to run tests for Grafana Enterprise using local environment.

To run the Enterprise tests with Grafana running in Docker Compose:

```sh
make testacc-enterprise-docker
```

Alternatively, you can use the `testacc-enterprise` target which will use your local `go`
installation:

```sh
# Assuming Grafana was run with:
# docker run --rm -p 3000:3000 grafana/grafana

GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
make testacc-enterprise
```

## Documentation

Documentation is generated with
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). Generated
files are in `docs/` and *should not be updated manually*. They are derived from:

- Schema `Description` fields in the provider Go code.
- [examples/](./examples)
- [templates/](./templates)

Use `go generate ./...` to update generated docs. This will be checked by CI on pull requests to ensure docs are in sync.

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
