Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.11 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-grafana`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-grafana
```

Build the provider in place:

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-grafana
$ make build
```

Install the provider as a Go binary (in `$GOPATH/bin`) and create a symlink to
it from the third-party user plugins directory in Terraform
(`~/.terraform.d/plugins/`):

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-grafana
$ make install
```

Developing the Provider
-----------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.11+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider in
current directory.

```sh
$ make build
==> Checking that code complies with gofmt requirements...
go build
$ ./terraform-provider-grafana
This binary is a plugin. These are not meant to be executed directly.
Please execute the program that consumes these plugins, which will
load any plugins automatically
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of acceptance tests on a local Grafana server,
run `make testacc`. This should be performed before merging or opening pull
requests.

```sh
$ GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin make testacc
```

This requires a running Grafana server locally. This provider targets
the latest version of Grafana, but older versions should be compatible where
possible. In some cases, older versions of this provider will work with
older versions of Grafana.

If you have [Docker](https://docs.docker.com/install/) installed, you can
run Grafana with the following command:

```sh
$ make test-serv
```

By default, this will use the latest version of Grafana based on their
Docker repository. You can specify the version with the following:

```sh
$ GRAFANA_VERSION=3.1.1 make test-serv
```

This command will run attached and will stop the Grafana server when
interrupted. Images will be cached locally by Docker so it is quicky to
restart the server as necessary. The server will use the default port and
credentials for the `GRAFANA_AUTH` and `GRAFANA_URL` environment variables.

Nightly acceptance tests are run against the `latest` tag of the Grafana
maintained Docker image.
