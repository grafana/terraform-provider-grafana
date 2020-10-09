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
# GRAFANA_VERSION=7.1.1 \
make test-serve

# In another...
GRAFANA_URL=http://localhost:3000 \
GRAFANA_AUTH=admin:admin \
make testacc
```

This codebase leverages
[grafana/grafana-api-golang-client](https://github.com/grafana/grafana-api-golang-client) as its Grafana API
client. All resources and data sources should leverage this.
