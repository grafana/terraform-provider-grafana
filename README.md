<a href="https://terraform.io">
  <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" title="Terraform" align="right" height="50" />
</a>

# Terraform Provider for Grafana

- Terraform website: https://terraform.io
- Grafana website: https://grafana.com
- Provider Documentation: https://registry.terraform.io/providers/grafana/grafana/latest/docs
- Terraform Chat: [Terraform Gitter](https://gitter.im/hashicorp-terraform/Lobby)
- Grafana Chat: [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT)
- Terraform Mailing List: [Google Groups](http://groups.google.com/group/terraform-tool)

## Development

If you're new to provider development, a good place to start is the [Extending
Terraform](https://www.terraform.io/docs/extend/index.html) docs.

Set up your local environment by installing [Go](http://www.golang.org). Also
recommended is [Docker](https://docs.docker.com/install/). Docker is not
required, but it makes running a local Grafana instance for acceptance tests
very easy.

Run [unit tests](https://www.terraform.io/docs/extend/testing/unit-testing.html):

```sh
make test
```

Run [acceptance tests](https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html):

```sh
# In one terminal, run a Grafana container.
# You may optionally override the image tag...
# GRAFANA_VERSION=7.3.4 \
make test-serve

# Alternatively, run Grafana with TLS:
# GRAFANA_VERSION=7.3.4 \
# make test-serve-tls

# In another...
GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
GRAFANA_ORG_ID=1 \
make testacc

# Alternatively, run acceptance tests with TLS client certs:
# GRAFANA_URL=https://localhost:3001 \
# GRAFANA_ORG_ID=1 \
# GRAFANA_AUTH=admin:admin \
# GRAFANA_TLS_KEY=$PWD/testdata/client.key \
# GRAFANA_TLS_CERT=$PWD/testdata/client.crt \
# GRAFANA_CA_CERT=$PWD/testdata/ca.crt \
# make testacc
```

This codebase leverages
[grafana/grafana-api-golang-client](https://github.com/grafana/grafana-api-golang-client) as its Grafana API
client. All resources and data sources should leverage this.

## Releasing

Builds and releases are automated with GitHub Actions and
[GoReleaser](https://github.com/goreleaser/goreleaser/). The changelog is
managed with
[github-changelog-generator](https://github.com/github-changelog-generator/github-changelog-generator).

Create a new release with the `release` Make target:

```
RELEASE_VERSION=v... \
CHANGELOG_GITHUB_TOKEN=... \
make release
```

Once the command exits, you can monitor the rest of the process on the [Actions
UI](https://github.com/grafana/terraform-provider-grafana/actions?query=workflow%3Arelease).

The Action creates the release, but leaves it in "draft" state. Open it up in a
[browser](https://github.com/grafana/terraform-provider-grafana/releases) and if
all looks well, mash the publish button.
