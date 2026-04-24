GRAFANA_VERSION ?= latest
DOCKER_COMPOSE_ARGS ?= --pull always --force-recreate --detach --remove-orphans --wait --renew-anon-volumes

# https://github.com/hashicorp/terraform-equivalence-testing — requires terraform on PATH,
# and GRAFANA_URL (and auth via GRAFANA_AUTH). If a test uses fixed identifiers, delete the 
# existing managed resource in Grafana or use a clean org before re-running
EQUIV_BIN ?= $(shell go env GOPATH)/bin/terraform-equivalence-testing

.PHONY: equivalence-test-install-tool equivalence-test-provider equivalence-test-update equivalence-test-diff

equivalence-test-install-tool:
	go install github.com/hashicorp/terraform-equivalence-testing@latest

equivalence-tests/generated.tfrc:
	printf '%s\n' \
	  'provider_installation {' \
	  '  filesystem_mirror {' \
	  "    path = \"$(CURDIR)/testdata/plugins\"" \
	  '    include = ["registry.terraform.io/grafana/grafana"]' \
	  '  }' \
	  '}' > $@

equivalence-test-provider:
	@mkdir -p testdata/plugins/registry.terraform.io/grafana/grafana/999.999.999/$$(go env GOOS)_$$(go env GOARCH)
	go build -o testdata/plugins/registry.terraform.io/grafana/grafana/999.999.999/$$(go env GOOS)_$$(go env GOARCH)/terraform-provider-grafana_v999.999.999_$$(go env GOOS)_$$(go env GOARCH) .

equivalence-test-update: equivalence-test-provider equivalence-tests/generated.tfrc
	@test -x "$(EQUIV_BIN)" || { echo "Install with: make equivalence-test-install-tool"; exit 1; }
	TF_CLI_CONFIG_FILE="$(CURDIR)/equivalence-tests/generated.tfrc" \
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" \
	"$(EQUIV_BIN)" update \
		--goldens="$(CURDIR)/equivalence-tests/goldens" \
		--tests="$(CURDIR)/equivalence-tests/tests"

equivalence-test-diff: equivalence-test-provider equivalence-tests/generated.tfrc
	@test -x "$(EQUIV_BIN)" || { echo "Install with: make equivalence-test-install-tool"; exit 1; }
	TF_CLI_CONFIG_FILE="$(CURDIR)/equivalence-tests/generated.tfrc" \
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" \
	"$(EQUIV_BIN)" diff \
		--goldens="$(CURDIR)/equivalence-tests/goldens" \
		--tests="$(CURDIR)/equivalence-tests/tests"

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
	export DOCKER_USER_UID="$(shell id -u)" && \
	export GRAFANA_URL=http://0.0.0.0:3000 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	make -C testdata generate && \
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
	@./scripts/release.sh

golangci-lint:
	docker run \
		--rm \
		--volume "$(shell pwd):/src" \
		--workdir "/src" \
		golangci/golangci-lint:v2.5.0 golangci-lint run ./... -v

docs:
	go generate ./...

codeowners:
	go run ./tools/codeowners > .github/CODEOWNERS

codeowners-check:
	go run ./tools/codeowners --check

linkcheck:
	docker run --rm --entrypoint sh -v "$$PWD:$$PWD" -w "$$PWD" python:3.11-alpine -c "pip3 install linkchecker && linkchecker --config .linkcheckerrc docs"

update-schema: ## Update provider schema only
	go build .
	./scripts/generate_schema.sh > provider_schema.json

generate-issue-template:
	go run scripts/generate_issue_template.go

generate-templates: ## Generate issue templates with schema update
	go run scripts/generate_issue_template.go --update-schema

generate-templates-quick: ## Generate issue templates (using cached schema)
	go run scripts/generate_issue_template.go
