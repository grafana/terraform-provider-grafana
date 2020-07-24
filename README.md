# Terraform Provider

- Website: https://terraform.io
- Documentation: https://www.terraform.io/docs/providers/grafana/index.html
- Chat: [Terraform Gitter](https://gitter.im/hashicorp-terraform/Lobby)
- Chat: [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT)
- Mailing List: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

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
[nytm/go-grafana-api](https://github.com/nytm/go-grafana-api) as its Grafana API
client. All resources and data sources should leverage this.
