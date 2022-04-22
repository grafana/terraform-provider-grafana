GRAFANA_VERSION ?= 8.4.4
OS_ARCH=darwin_arm64
BINARY=terraform-provider-${NAME}
HOSTNAME=grafana.com
NAMESPACE=raintank
NAME=grafana
VERSION=0.2.4

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

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

