GRAFANA_VERSION ?= 8.4.3

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

testacc-oss: 
	TF_ACC_OSS=true make testacc

testacc-enterprise:
	TF_ACC_ENTERPRISE=true make testacc

testacc-cloud:
	TF_ACC_CLOUD=true make testacc

testacc-docker:
	GRAFANA_VERSION=$(GRAFANA_VERSION) \
		docker-compose \
		-f ./docker-compose.yml \
		run --rm -e TESTARGS="$(TESTARGS)" \
		grafana-provider \
		make testacc-oss

testacc-docker-tls:
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

drone:
	drone jsonnet --stream --source .drone/drone.jsonnet --target .drone/drone.yml --format
	drone lint .drone/drone.yml
	drone sign --save grafana/terraform-provider-grafana .drone/drone.yml
