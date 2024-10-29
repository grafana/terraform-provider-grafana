GRAFANA_VERSION ?= 11.0.0
DOCKER_COMPOSE_ARGS ?= --force-recreate --detach --remove-orphans --wait --renew-anon-volumes

testacc:
	go build -o testdata/plugins/registry.terraform.io/grafana/grafana/999.999.999/$$(go env GOOS)_$$(go env GOARCH)/terraform-provider-grafana_v999.999.999_$$(go env GOOS)_$$(go env GOARCH) .
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Test OSS features
testacc-oss:
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" TF_ACC_OSS=true make testacc

# Test Enterprise features
testacc-enterprise:
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" TF_ACC_ENTERPRISE=true make testacc

# Test Cloud API features
testacc-cloud-api:
	TF_ACC_CLOUD_API=true make testacc

# Test Cloud instance features (ex: Machine Learning and Synthetic Monitoring)
testacc-cloud-instance:
	TF_ACC_CLOUD_INSTANCE=true make testacc

testacc-oss-docker:
	export GRAFANA_URL=http://0.0.0.0:3000 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	docker compose up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-oss && \
	docker compose down

testacc-enterprise-docker:
	export GRAFANA_URL=http://0.0.0.0:3000 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	GRAFANA_IMAGE=grafana/grafana-enterprise docker compose up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-enterprise && \
	docker compose down

testacc-tls-docker:
	export GRAFANA_URL=https://0.0.0.0:3001 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	make -C testdata generate && \
	docker compose --profile tls up $(DOCKER_COMPOSE_ARGS) && \
	GRAFANA_TLS_KEY=$$(pwd)/testdata/client.key GRAFANA_TLS_CERT=$$(pwd)/testdata/client.crt GRAFANA_CA_CERT=$$(pwd)/testdata/ca.crt make testacc-oss && \
	docker compose --profile tls down

testacc-subpath-docker:
	export GRAFANA_SUBPATH=/grafana/ && \
	export GF_SERVER_SERVE_FROM_SUB_PATH=true && \
	export GRAFANA_URL=http://0.0.0.0:3001$${GRAFANA_SUBPATH} && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	docker compose --profile proxy up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-oss && \
	docker compose --profile proxy down

integration-test:
	DOCKER_COMPOSE_ARGS="$(DOCKER_COMPOSE_ARGS)" GRAFANA_VERSION=$(GRAFANA_VERSION) ./testdata/integration/test.sh

release:
	@test $${RELEASE_VERSION?Please set environment variable RELEASE_VERSION}
	@git tag $$RELEASE_VERSION
	@git push origin $$RELEASE_VERSION

golangci-lint:
	docker run \
		--rm \
		--volume "$(shell pwd):/src" \
		--workdir "/src" \
		golangci/golangci-lint:v1.61.0 golangci-lint run ./... -v

docs:
	go generate ./...

linkcheck:
	docker run --rm --entrypoint sh -v "$$PWD:$$PWD" -w "$$PWD" python:3.11-alpine -c "pip3 install linkchecker && linkchecker --config .linkcheckerrc docs"
