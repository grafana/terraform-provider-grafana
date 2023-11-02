GRAFANA_VERSION ?= 10.1.5

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m -parallel 4

# Test OSS features
testacc-oss:
	TF_ACC_OSS=true make testacc

# Test Enterprise features
testacc-enterprise:
	TF_ACC_ENTERPRISE=true make testacc

# Test Cloud API features
testacc-cloud-api:
	TF_ACC_CLOUD_API=true make testacc

# Test Cloud instance features (ex: Machine Learning and Synthetic Monitoring)
testacc-cloud-instance:
	TF_ACC_CLOUD_INSTANCE=true make testacc

testacc-oss-docker:
	GRAFANA_VERSION=$(GRAFANA_VERSION) docker compose up --force-recreate --detach --remove-orphans --wait

	GRAFANA_VERSION=$(GRAFANA_VERSION) \
	GRAFANA_URL="http://$$(docker compose port grafana 3000)" \
	GRAFANA_AUTH="admin:admin" \
	make testacc-oss

	docker compose down

testacc-enterprise-docker:
	GRAFANA_IMAGE=grafana/grafana-enterprise GRAFANA_VERSION=$(GRAFANA_VERSION) docker compose up --force-recreate --detach --remove-orphans --wait

	GRAFANA_VERSION=$(GRAFANA_VERSION) \
	GRAFANA_URL="http://$$(docker compose port grafana 3000)" \
	GRAFANA_AUTH="admin:admin" \
	make testacc-enterprise

	docker compose down

testacc-oss-docker-tls:
	make -C testdata generate
	GRAFANA_VERSION=$(GRAFANA_VERSION) docker compose --profile tls up --force-recreate --detach --remove-orphans --wait

	GRAFANA_VERSION=$(GRAFANA_VERSION) \
	GRAFANA_URL="https://$$(docker compose port mtls-proxy 3001)" \
	GRAFANA_AUTH="admin:admin" \
	GRAFANA_TLS_KEY=$$(pwd)/testdata/client.key \
    GRAFANA_TLS_CERT=$$(pwd)/testdata/client.crt \
    GRAFANA_CA_CERT=$$(pwd)/testdata/ca.crt \
	make testacc-oss

	docker compose --profile tls down

release:
	@test $${RELEASE_VERSION?Please set environment variable RELEASE_VERSION}
	@git tag $$RELEASE_VERSION
	@git push origin $$RELEASE_VERSION

golangci-lint:
	docker run \
		--rm \
		--volume "$(shell pwd):/src" \
		--workdir "/src" \
		golangci/golangci-lint:v1.54 golangci-lint run ./... -v

linkcheck:
	docker run --rm --entrypoint sh -v "$$PWD:$$PWD" -w "$$PWD" python:3.11-alpine -c "pip3 install linkchecker && linkchecker --config .linkcheckerrc docs"
