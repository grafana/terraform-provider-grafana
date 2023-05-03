GRAFANA_VERSION ?= 9.5.1

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

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

testacc-docker:
	make -C testdata generate
	docker-compose -f ./docker-compose.yml stop
	GRAFANA_VERSION=$(GRAFANA_VERSION) \
		docker-compose \
		-f ./docker-compose.yml \
		run --rm -e TESTARGS="$(TESTARGS)" \
		grafana-provider \
		make testacc-oss

testacc-docker-tls:
	make -C testdata generate
	docker-compose -f ./docker-compose.yml -f ./docker-compose.tls.yml stop 
	GRAFANA_VERSION=$(GRAFANA_VERSION) \
		docker-compose \
		-f ./docker-compose.yml \
		-f ./docker-compose.tls.yml \
		run --rm -e TESTARGS="$(TESTARGS)" \
		grafana-provider \
		make testacc-oss

release:
	@test $${RELEASE_VERSION?Please set environment variable RELEASE_VERSION}
	@git tag $$RELEASE_VERSION
	@git push origin $$RELEASE_VERSION

DRONE_DOCKER := docker run --rm -e DRONE_SERVER -e DRONE_TOKEN -v ${PWD}:${PWD} -w "${PWD}" drone/cli:1.6.1
drone:
	$(DRONE_DOCKER) jsonnet --stream --source .drone/drone.jsonnet --target .drone/drone.yml --format
	$(DRONE_DOCKER) lint .drone/drone.yml
	$(DRONE_DOCKER) sign --save grafana/terraform-provider-grafana .drone/drone.yml

golangci-lint:
	docker run \
		--rm \
		--volume "$(shell pwd):/src" \
		--workdir "/src" \
		golangci/golangci-lint:v1.52 golangci-lint run ./...

linkcheck:
	docker run -it --entrypoint sh -v "$$PWD:$$PWD" -w "$$PWD" python:3.11-alpine -c "pip3 install linkchecker && linkchecker --config .linkcheckerrc docs"

